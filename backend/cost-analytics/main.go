package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type CostAnalyticsService struct {
	pg       *sql.DB
	ch       driver.Conn
	kafkaRdr *kafka.Reader
}

type CostSummary struct {
	TotalCost     float64 `json:"total_cost"`
	AmortizedCost float64 `json:"amortized_cost"`
	ListCost      float64 `json:"list_cost"`
	ServiceCount  uint64  `json:"service_count"`
	ResourceCount uint64  `json:"resource_count"`
}

type KPIData struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	Trend  float64 `json:"trend"`
	Target float64 `json:"target,omitempty"`
	Status string  `json:"status"`
}

type Budget struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Amount         float64   `json:"amount"`
	Spent          float64   `json:"spent"`
	Remaining      float64   `json:"remaining"`
	Period         string    `json:"period"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	AlertThreshold float64   `json:"alert_threshold"`
	Provider       string    `json:"provider"`
	AccountID      string    `json:"account_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type FocusRecord struct {
	Provider           string            `json:"provider"`
	BillingAccountID   string            `json:"billing_account_id"`
	ServiceName        string            `json:"service_name"`
	ResourceType       string            `json:"resource_type"`
	ResourceID         string            `json:"resource_id"`
	Region             string            `json:"region"`
	UsageQuantity      float64           `json:"usage_quantity"`
	UsageUnit          string            `json:"usage_unit"`
	EffectiveCost      float64           `json:"effective_cost"`
	EffectiveCostBRL   float64           `json:"effective_cost_brl"`
	ListCost           float64           `json:"list_cost"`
	ListCostBRL        float64           `json:"list_cost_brl"`
	ContractedCost     float64           `json:"contracted_cost"`
	ContractedCostBRL  float64           `json:"contracted_cost_brl"`
	AmortizedCost      float64           `json:"amortized_cost"`
	AmortizedCostBRL   float64           `json:"amortized_cost_brl"`
	Date               string            `json:"date"`
	Environment        string            `json:"environment"`
	Application        string            `json:"application"`
	BusinessUnit       string            `json:"business_unit"`
	Tags               map[string]string `json:"tags"`
}

func main() {
	pgConn := getEnv("POSTGRES_DSN", "postgres://finops:finops@postgres:5432/finops?sslmode=disable")
	pg, err := sql.Open("postgres", pgConn)
	if err != nil {
		log.Fatal(err)
	}
	pg.SetMaxOpenConns(25)
	pg.SetMaxIdleConns(10)
	pg.SetConnMaxLifetime(5 * time.Minute)

	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{getEnv("CLICKHOUSE_ADDR", "clickhouse:9000")},
		Auth: clickhouse.Auth{
			Database: getEnv("CLICKHOUSE_DB", "finops"),
			Username: getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := ensureClickHouseSchema(chConn); err != nil {
		log.Fatalf("Failed to ensure ClickHouse schema: %v", err)
	}

	kafkaRdr := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{getEnv("KAFKA_BROKERS", "kafka:9092")},
		Topic:       "cost.raw",
		GroupID:     "cost-analytics",
		StartOffset: kafka.LastOffset,
		MaxWait:     1 * time.Second,
		MinBytes:    1e3,
		MaxBytes:    10e6,
		MaxAttempts: 3,
	})

	svc := &CostAnalyticsService{pg: pg, ch: chConn, kafkaRdr: kafkaRdr}
	go svc.consumeKafka()
	go svc.runDailyAggregator()

	r := gin.Default()
	r.GET("/health", healthHandler)
	r.HEAD("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.HEAD("/ready", readyHandler)
	r.GET("/live", liveHandler)
	r.HEAD("/live", liveHandler)
	r.GET("/metrics", metricsHandler)

	api := r.Group("/api/v1")
	api.GET("/costs", svc.getCosts)
	api.GET("/costs/trends", svc.getTrends)
	api.GET("/costs/services", svc.getByService)
	api.GET("/costs/applications", svc.getByApplication)
	api.GET("/costs/environments", svc.getByEnvironment)
	api.GET("/costs/business-units", svc.getByBusinessUnit)
	api.GET("/costs/regions", svc.getByRegion)
	api.GET("/costs/providers", svc.getByProvider)
	api.GET("/kpis", svc.getKPIs)
	api.GET("/dashboard/executive", svc.getExecutiveDashboard)
	api.GET("/dashboard/operational", svc.getOperationalDashboard)
	api.GET("/anomalies", svc.getAnomalies)
	api.GET("/accounts", svc.getAccounts)
	api.GET("/forecast", svc.getForecast)
	api.GET("/recommendations", svc.getRecommendations)
	api.POST("/recommendations/:id/apply", svc.applyRecommendation)

	api.GET("/budgets", svc.getBudgets)
	api.POST("/budgets", svc.createBudget)
	api.PUT("/budgets/:id", svc.updateBudget)
	api.DELETE("/budgets/:id", svc.deleteBudget)

	api.GET("/admin/users", svc.getUsers)
	api.POST("/admin/users", svc.createUser)
	api.PUT("/admin/users/:id", svc.updateUser)
	api.DELETE("/admin/users/:id", svc.deleteUser)
	api.GET("/admin/audit", svc.getAudit)
	api.POST("/admin/truncate-dates", svc.truncateDates)

	port := getEnv("PORT", "8082")
	log.Printf("Cost Analytics starting on port %s", port)
	r.Run(":" + port)
}

func ensureClickHouseSchema(ch driver.Conn) error {
	ctx := context.Background()

	// Ensure database exists. Use the configured connection first; if it fails,
	// open a temporary connection to the 'default' database to create it.
	if err := ch.Exec(ctx, "CREATE DATABASE IF NOT EXISTS finops"); err != nil {
		log.Printf("[SCHEMA] Could not create database with configured connection: %v", err)
		log.Printf("[SCHEMA] Retrying with 'default' database...")
		tmpCh, err := clickhouse.Open(&clickhouse.Options{
			Addr: []string{getEnv("CLICKHOUSE_ADDR", "clickhouse:9000")},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: getEnv("CLICKHOUSE_USER", "default"),
				Password: getEnv("CLICKHOUSE_PASSWORD", ""),
			},
		})
		if err != nil {
			return fmt.Errorf("failed to open temporary ClickHouse connection: %w", err)
		}
		defer tmpCh.Close()
		if err := tmpCh.Exec(ctx, "CREATE DATABASE IF NOT EXISTS finops"); err != nil {
			return fmt.Errorf("failed to create database finops: %w", err)
		}
	}

	schemaDDL := `
CREATE TABLE IF NOT EXISTS finops.costs_raw (
    provider String,
    billing_account_id String,
    service_name String,
    resource_type String,
    resource_id String,
    region String,
    usage_quantity Float64,
    usage_unit String,
    effective_cost Float64,
    effective_cost_brl Float64,
    list_cost Float64,
    list_cost_brl Float64,
    contracted_cost Float64,
    contracted_cost_brl Float64,
    amortized_cost Float64,
    amortized_cost_brl Float64,
    date Date,
    environment String,
    application String,
    business_unit String,
    tags Map(String, String)
) ENGINE = MergeTree()
ORDER BY (provider, date, service_name)
PARTITION BY toYYYYMM(date)
TTL date + INTERVAL 36 MONTH;

CREATE TABLE IF NOT EXISTS finops.costs_daily (
    provider String,
    service_name String,
    region String,
    date Date,
    total_cost Float64,
    total_cost_brl Float64,
    total_usage Float64,
    environment String,
    application String,
    business_unit String
) ENGINE = MergeTree()
ORDER BY (provider, date, service_name)
PARTITION BY toYYYYMM(date);

CREATE TABLE IF NOT EXISTS finops.anomalies (
    id UUID DEFAULT generateUUIDv4(),
    service_name String,
    region String,
    date Date,
    expected_value Float64,
    actual_value Float64,
    deviation Float64,
    severity String,
    detected_at DateTime
) ENGINE = MergeTree()
ORDER BY (date, service_name);

CREATE TABLE IF NOT EXISTS finops.forecasts (
    service_name String,
    region String,
    forecast_date Date,
    predicted_cost Float64,
    predicted_cost_brl Float64,
    confidence_lower Float64,
    confidence_upper Float64,
    model_version String,
    created_at DateTime
) ENGINE = MergeTree()
ORDER BY (forecast_date, service_name);

CREATE TABLE IF NOT EXISTS finops.recommendations (
    id UUID DEFAULT generateUUIDv4(),
    category String,
    title String,
    description String,
    potential_savings Float64,
    potential_savings_brl Float64,
    priority String,
    status String,
    created_at DateTime
) ENGINE = MergeTree()
ORDER BY (created_at, category);
`
	statements := strings.Split(schemaDDL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := ch.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("failed to apply ClickHouse schema statement: %w\nstatement: %s", err, stmt)
		}
	}
	log.Printf("[SCHEMA] ClickHouse schema ensured")
	return nil
}

func (s *CostAnalyticsService) consumeKafka() {
	ctx := context.Background()
	const batchSize = 500
	const flushInterval = 5 * time.Second

	batch := make([]kafka.Message, 0, batchSize)
	records := make([]FocusRecord, 0, batchSize)
	flushTimer := time.NewTimer(flushInterval)
	defer flushTimer.Stop()

	for {
		select {
		case <-flushTimer.C:
			if len(batch) > 0 {
				s.flushKafkaBatch(ctx, &batch, &records)
			}
			flushTimer.Reset(flushInterval)
		default:
			msg, err := s.kafkaRdr.ReadMessage(ctx)
			if err != nil {
				log.Printf("[KAFKA] Read error: %v", err)
				continue
			}

			var rec FocusRecord
			if err := json.Unmarshal(msg.Value, &rec); err != nil {
				log.Printf("[KAFKA] Unmarshal error: %v", err)
				if cErr := s.kafkaRdr.CommitMessages(ctx, msg); cErr != nil {
					log.Printf("[KAFKA] Commit error: %v", cErr)
				}
				continue
			}

			batch = append(batch, msg)
			records = append(records, rec)

			if len(batch) >= batchSize {
				s.flushKafkaBatch(ctx, &batch, &records)
				flushTimer.Reset(flushInterval)
			}
		}
	}
}

func (s *CostAnalyticsService) flushKafkaBatch(ctx context.Context, batch *[]kafka.Message, records *[]FocusRecord) {
	if len(*records) == 0 {
		return
	}

	if err := s.insertRawCostBatch(ctx, *records); err != nil {
		log.Printf("[KAFKA] ClickHouse batch insert error: %v", err)
		return
	}

	if err := s.kafkaRdr.CommitMessages(ctx, *batch...); err != nil {
		log.Printf("[KAFKA] Commit error: %v", err)
		return
	}

	log.Printf("[KAFKA] Persisted batch of %d records", len(*records))
	*batch = (*batch)[:0]
	*records = (*records)[:0]
}

func (s *CostAnalyticsService) insertRawCostBatch(ctx context.Context, records []FocusRecord) error {
	batch, err := s.ch.PrepareBatch(ctx, "INSERT INTO costs_raw")
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, rec := range records {
		if rec.Tags == nil {
			rec.Tags = map[string]string{}
		}
		// Normalize tag values into dedicated columns if they are empty.
		if rec.Environment == "" && rec.Tags["environment"] != "" {
			rec.Environment = rec.Tags["environment"]
		}
		if rec.Application == "" && rec.Tags["application"] != "" {
			rec.Application = rec.Tags["application"]
		}
		if rec.BusinessUnit == "" && rec.Tags["business_unit"] != "" {
			rec.BusinessUnit = rec.Tags["business_unit"]
		}
		// Build tags in a fixed order so duplicates are detected correctly by ClickHouse.
		normalizedTags := map[string]string{
			"application":   rec.Application,
			"business_unit": rec.BusinessUnit,
			"environment":   rec.Environment,
		}
		// Recalculate BRL columns deterministically to avoid floating-point drift
		// that breaks duplicate detection.
		rate := getEnvFloat("USD_TO_BRL_RATE", 5.15)
		if err := batch.Append(
			rec.Provider, rec.BillingAccountID, rec.ServiceName, rec.ResourceType, rec.ResourceID, rec.Region,
			rec.UsageQuantity, rec.UsageUnit,
			rec.EffectiveCost, math.Round(rec.EffectiveCost*rate*100)/100,
			rec.ListCost, math.Round(rec.ListCost*rate*100)/100,
			rec.ContractedCost, math.Round(rec.ContractedCost*rate*100)/100,
			rec.AmortizedCost, math.Round(rec.AmortizedCost*rate*100)/100,
			rec.Date, rec.Environment, rec.Application, rec.BusinessUnit, normalizedTags,
		); err != nil {
			return fmt.Errorf("append batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}
	return nil
}

func (s *CostAnalyticsService) runDailyAggregator() {
	// Initial delay then every 2 minutes
	time.Sleep(30 * time.Second)
	s.aggregateDaily()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.aggregateDaily()
	}
}

func (s *CostAnalyticsService) aggregateDaily() {
	query := `
		INSERT INTO costs_daily
		SELECT
			provider,
			service_name,
			region,
			date,
			sum(effective_cost),
			sum(effective_cost_brl),
			sum(usage_quantity),
			environment,
			application,
			business_unit
		FROM costs_raw
		WHERE date >= today() - 90
		GROUP BY provider, service_name, region, date, environment, application, business_unit
	`
	if err := s.ch.Exec(context.Background(), query); err != nil {
		log.Printf("[AGGREGATOR] Daily aggregation error: %v", err)
	} else {
		log.Printf("[AGGREGATOR] Daily costs refreshed")
	}
}

func currencyColumn(c *gin.Context, usdCol, brlCol string) string {
	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	if currency == "BRL" {
		return brlCol
	}
	return usdCol
}

func dateRangeFromQuery(c *gin.Context) (start string, end string) {
	start = c.Query("start_date")
	end = c.Query("end_date")
	if start == "" || end == "" {
		start = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
		end = time.Now().Format("2006-01-02")
	}
	return
}

func dateWhereClause(start, end string) string {
	return fmt.Sprintf("date BETWEEN '%s' AND '%s'", start, end)
}

func (s *CostAnalyticsService) getCosts(c *gin.Context) {
	start := c.Query("start_date")
	end := c.Query("end_date")
	provider := c.Query("provider")
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	amortizedCol := currencyColumn(c, "amortized_cost", "amortized_cost_brl")
	listCol := currencyColumn(c, "list_cost", "list_cost_brl")

	if start == "" || end == "" {
		start = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
		end = time.Now().Format("2006-01-02")
	}

	query := fmt.Sprintf(`SELECT sum(%s), sum(%s), sum(%s), uniqExact(service_name), uniqExact(resource_id) FROM costs_raw WHERE date BETWEEN ? AND ?`, costCol, amortizedCol, listCol)
	args := []interface{}{start, end}
	if provider != "" {
		query += " AND provider = ?"
		args = append(args, provider)
	}

	row := s.ch.QueryRow(context.Background(), query, args...)
	var summary CostSummary
	var totalCost, amortizedCost, listCost *float64
	var serviceCount, resourceCount *uint64
	if err := row.Scan(&totalCost, &amortizedCost, &listCost, &serviceCount, &resourceCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	summary.TotalCost = derefFloat(totalCost)
	summary.AmortizedCost = derefFloat(amortizedCost)
	summary.ListCost = derefFloat(listCost)
	summary.ServiceCount = derefUint64(serviceCount)
	summary.ResourceCount = derefUint64(resourceCount)
	c.JSON(http.StatusOK, summary)
}

func derefFloat(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0
}

func derefUint64(v *uint64) uint64 {
	if v != nil {
		return *v
	}
	return 0
}

func (s *CostAnalyticsService) getTrends(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT date, sum(%s) as cost, sum(usage_quantity) as usage FROM costs_raw WHERE %s GROUP BY date ORDER BY date`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()

	var results []gin.H
	for rows.Next() {
		var date time.Time
		var cost, usage float64
		if err := rows.Scan(&date, &cost, &usage); err != nil {
			log.Printf("[TRENDS] Scan error: %v", err)
			continue
		}
		results = append(results, gin.H{"date": date.Format("2006-01-02"), "cost": cost, "usage": usage})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByService(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT service_name, sum(%s) as cost, sum(usage_quantity) as usage FROM costs_raw WHERE %s GROUP BY service_name ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var svc string
		var cost, usage float64
		rows.Scan(&svc, &cost, &usage)
		results = append(results, gin.H{"service": svc, "cost": cost, "usage": usage})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByApplication(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT application, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY application ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var app string
		var cost float64
		rows.Scan(&app, &cost)
		results = append(results, gin.H{"application": app, "cost": cost})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByEnvironment(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT environment, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY environment ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var env string
		var cost float64
		rows.Scan(&env, &cost)
		results = append(results, gin.H{"environment": env, "cost": cost})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByBusinessUnit(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT business_unit, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY business_unit ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var bu string
		var cost float64
		rows.Scan(&bu, &cost)
		results = append(results, gin.H{"business_unit": bu, "cost": cost})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByRegion(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT region, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY region ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var region string
		var cost float64
		rows.Scan(&region, &cost)
		results = append(results, gin.H{"region": region, "cost": cost})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByProvider(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT provider, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY provider ORDER BY cost DESC`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}
	defer rows.Close()
	var results []gin.H
	for rows.Next() {
		var provider string
		var cost float64
		rows.Scan(&provider, &cost)
		results = append(results, gin.H{"provider": provider, "cost": cost})
	}
	if results == nil {
		results = []gin.H{}
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getKPIs(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	listCostCol := currencyColumn(c, "list_cost", "list_cost_brl")

	total := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", costCol, dateWhereClause(start, end)))
	listTotal := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", listCostCol, dateWhereClause(start, end)))

	// Previous period of same duration
	startT, _ := time.Parse("2006-01-02", start)
	endT, _ := time.Parse("2006-01-02", end)
	duration := endT.Sub(startT)
	prevStart := startT.Add(-duration - 24*time.Hour).Format("2006-01-02")
	prevEnd := endT.Add(-duration - 24*time.Hour).Format("2006-01-02")
	previous := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", costCol, dateWhereClause(prevStart, prevEnd)))

	trend := 0.0
	if previous > 0 {
		trend = (total - previous) / previous
	}
	forecast := total
	if total > 0 {
		forecast = total * (1 + trend)
	}

	// Cost efficiency: how close paid cost is to list cost (1 = optimal, paid = list)
	efficiency := 1.0
	if listTotal > 0 {
		efficiency = total / listTotal
	}

	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	budgetUtil := s.computeBudgetUtilization(costCol)
	kpis := []KPIData{
		{Name: "Total Cost", Value: total, Unit: currency, Trend: trend, Status: statusForTrend(trend)},
		{Name: "Forecast (next period)", Value: forecast, Unit: currency, Trend: trend, Status: statusForTrend(trend)},
		{Name: "Cost Efficiency", Value: efficiency, Unit: "ratio", Trend: 0, Target: 0.90, Status: statusForValue(efficiency, 0.90)},
		{Name: "Budget Utilization", Value: budgetUtil, Unit: "ratio", Trend: 0, Target: 0.80, Status: statusForValue(1-budgetUtil, 0.20)},
	}
	c.JSON(http.StatusOK, kpis)
}

func (s *CostAnalyticsService) sumCost(query string, args ...interface{}) float64 {
	row := s.ch.QueryRow(context.Background(), query, args...)
	var v *float64
	row.Scan(&v)
	return derefFloat(v)
}

func statusForValue(value, target float64) string {
	if value < target {
		return "warning"
	}
	return "good"
}

func statusForTrend(trend float64) string {
	if trend > 0.05 {
		return "warning"
	}
	return "good"
}

func (s *CostAnalyticsService) getExecutiveDashboard(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	listCostCol := currencyColumn(c, "list_cost", "list_cost_brl")
	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))

	totalCost := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", costCol, dateWhereClause(start, end)))
	listCost := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", listCostCol, dateWhereClause(start, end)))

	topServices, _ := s.queryTop(fmt.Sprintf("SELECT service_name, sum(%s) FROM costs_raw WHERE %s GROUP BY service_name ORDER BY sum(%s) DESC LIMIT 5", costCol, dateWhereClause(start, end), costCol))
	topApps, _ := s.queryTop(fmt.Sprintf("SELECT application, sum(%s) FROM costs_raw WHERE %s GROUP BY application ORDER BY sum(%s) DESC LIMIT 5", costCol, dateWhereClause(start, end), costCol))

	// Forecast based on actual trend
	startT, _ := time.Parse("2006-01-02", start)
	endT, _ := time.Parse("2006-01-02", end)
	duration := endT.Sub(startT)
	prevStart := startT.Add(-duration - 24*time.Hour).Format("2006-01-02")
	prevEnd := endT.Add(-duration - 24*time.Hour).Format("2006-01-02")
	previousCost := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", costCol, dateWhereClause(prevStart, prevEnd)))
	forecast := totalCost
	if previousCost > 0 {
		forecast = totalCost * (1 + (totalCost-previousCost)/previousCost)
	}

	// Potential savings: difference between list cost and effective cost
	potentialSavings := math.Max(0, listCost-totalCost)

	c.JSON(http.StatusOK, gin.H{
		"total_cost":        totalCost,
		"forecast_30d":      forecast,
		"potential_savings": potentialSavings,
		"top_services":      topServices,
		"top_applications":  topApps,
		"currency":          currency,
		"kpis": []gin.H{
			{"name": "Total Cost", "value": totalCost},
			{"name": "Forecast", "value": forecast},
			{"name": "Cost Efficiency", "value": s.computeCostEfficiency(costCol, listCostCol)},
			{"name": "Budget Utilization", "value": s.computeBudgetUtilization(costCol)},
		},
	})
}

func (s *CostAnalyticsService) computeCostEfficiency(costCol, listCostCol string) float64 {
	total := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", costCol))
	list := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", listCostCol))
	if list > 0 {
		return total / list
	}
	return 1
}

func (s *CostAnalyticsService) getAnomalies(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT date, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY date ORDER BY date`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type point struct {
		date time.Time
		cost float64
	}
	var points []point
	var total float64
	for rows.Next() {
		var p point
		if err := rows.Scan(&p.date, &p.cost); err != nil {
			log.Printf("[ANOMALIES] Scan error: %v", err)
			continue
		}
		points = append(points, p)
		total += p.cost
	}

	var mean, std float64
	if len(points) > 0 {
		mean = total / float64(len(points))
	}
	for _, p := range points {
		std += (p.cost - mean) * (p.cost - mean)
	}
	if len(points) > 1 {
		std = math.Sqrt(std / float64(len(points)-1))
	}

	var anomalies []gin.H
	for _, p := range points {
		if std == 0 {
			continue
		}
		z := (p.cost - mean) / std
		if math.Abs(z) > 2.0 {
			severity := "medium"
			if math.Abs(z) > 3.0 {
				severity = "high"
			}
			anomalies = append(anomalies, gin.H{
				"date":       p.date.Format("2006-01-02"),
				"value":      p.cost,
				"expected":   mean,
				"deviation":  z,
				"severity":   severity,
				"score":      math.Abs(z),
				"method":     "z-score",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"provider":        c.Query("provider"),
		"account_id":      c.Query("account_id"),
		"anomalies":       anomalies,
		"total_anomalies": len(anomalies),
		"total_impact":    total,
		"baseline_mean":   mean,
		"baseline_std":    std,
		"generated_at":    time.Now().UTC(),
	})
}

func (s *CostAnalyticsService) getForecast(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")

	// Historical daily costs
	query := fmt.Sprintf(`SELECT date, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY date ORDER BY date`, costCol, dateWhereClause(start, end))
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type point struct {
		date  time.Time
		value float64
	}
	var history []point
	var total float64
	for rows.Next() {
		var p point
		if err := rows.Scan(&p.date, &p.value); err != nil {
			log.Printf("[FORECAST] Scan error: %v", err)
			continue
		}
		history = append(history, p)
		total += p.value
	}

	// Simple linear trend forecast
	n := float64(len(history))
	var sumX, sumY, sumXY, sumXX float64
	for i, p := range history {
		x := float64(i)
		y := p.value
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}
	slope := 0.0
	intercept := 0.0
	if n > 1 && (n*sumXX-sumX*sumX) != 0 {
		slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
		intercept = (sumY - slope*sumX) / n
	} else if n > 0 {
		intercept = sumY / n
	}

	forecastDays := 30
	if d := c.Query("forecast_days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && v > 0 {
			forecastDays = v
		}
	}

	lastDate := time.Now()
	if len(history) > 0 {
		lastDate = history[len(history)-1].date
	}

	var forecast []gin.H
	var totalForecast float64
	std := 0.0
	if len(history) > 1 {
		mean := total / n
		var ss float64
		for _, p := range history {
			ss += (p.value - mean) * (p.value - mean)
		}
		std = math.Sqrt(ss / (n - 1))
	}
	for i := 1; i <= forecastDays; i++ {
		x := n + float64(i) - 1
		predicted := intercept + slope*x
		if predicted < 0 {
			predicted = 0
		}
		d := lastDate.AddDate(0, 0, i).Format("2006-01-02")
		forecast = append(forecast, gin.H{
			"date":   d,
			"value":  predicted,
			"lower":  math.Max(0, predicted-std),
			"upper":  predicted + std,
		})
		totalForecast += predicted
	}

	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	c.JSON(http.StatusOK, gin.H{
		"provider":       c.Query("provider"),
		"account_id":     c.Query("account_id"),
		"period":         fmt.Sprintf("%dd", forecastDays),
		"model":          "Linear Trend",
		"forecast":       forecast,
		"total_forecast": totalForecast,
		"trend":          slope,
		"confidence":     0.85,
		"generated_at":   time.Now().UTC(),
		"currency":       currency,
	})
}

func (s *CostAnalyticsService) getRecommendations(c *gin.Context) {
	start, end := dateRangeFromQuery(c)
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")

	totalCost := s.sumCost(fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE %s", costCol, dateWhereClause(start, end)))

	// Top services by cost
	serviceRows, err := s.ch.Query(context.Background(), fmt.Sprintf(`SELECT service_name, sum(%s) as cost FROM costs_raw WHERE %s GROUP BY service_name ORDER BY cost DESC LIMIT 10`, costCol, dateWhereClause(start, end)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer serviceRows.Close()

	var recommendations []gin.H
	idx := 1
	for serviceRows.Next() {
		var svc string
		var cost float64
		serviceRows.Scan(&svc, &cost)
		if svc == "" {
			continue
		}
		pct := 0.0
		if totalCost > 0 {
			pct = (cost / totalCost) * 100
		}
		if pct < 5 {
			continue
		}
		savings := cost * 0.1
		recommendations = append(recommendations, gin.H{
			"id":                  fmt.Sprintf("rec-%d", idx),
			"category":            "savings",
			"title":               fmt.Sprintf("Revisar custos do serviço %s", svc),
			"description":         fmt.Sprintf("O serviço %s representa %.1f%% do custo total no período. Avalie oportunidades de otimização.", svc, pct),
			"resource_id":         "",
			"resource_type":       "service",
			"service":             svc,
			"region":              "",
			"current_cost":        cost,
			"projected_cost":      cost * 0.9,
			"savings":             savings,
			"savings_percentage":  10.0,
			"confidence":          0.75,
			"priority":            "medium",
			"justification":       fmt.Sprintf("Custo mensal de %s no período selecionado.", svc),
			"action":              "Analisar uso do serviço e negociar descontos ou ajustar configurações.",
			"risk":                "Baixo - revisão apenas.",
			"implementation":      "Reunião com time de finanças e engenharia.",
			"created_at":          time.Now().UTC(),
		})
		idx++
	}

	totalSavings := 0.0
	for _, r := range recommendations {
		if v, ok := r["savings"].(float64); ok {
			totalSavings += v
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"provider":             c.Query("provider"),
		"account_id":           c.Query("account_id"),
		"total_savings":        totalSavings,
		"total_opportunities":  len(recommendations),
		"recommendations":      recommendations,
		"generated_at":         time.Now().UTC(),
	})
}

func (s *CostAnalyticsService) applyRecommendation(c *gin.Context) {
	id := c.Param("id")
	// In a real system, this would trigger a workflow. Here we just acknowledge.
	c.JSON(http.StatusOK, gin.H{"status": "applied", "id": id, "applied_at": time.Now().UTC()})
}

func (s *CostAnalyticsService) getOperationalDashboard(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")

	byService, _ := s.queryTop(fmt.Sprintf("SELECT service_name, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY service_name ORDER BY sum(%s) DESC LIMIT 10", costCol, costCol))
	byEnv, _ := s.queryTop(fmt.Sprintf("SELECT environment, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY environment ORDER BY sum(%s) DESC LIMIT 10", costCol, costCol))
	byRegion, _ := s.queryTop(fmt.Sprintf("SELECT region, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY region ORDER BY sum(%s) DESC LIMIT 10", costCol, costCol))
	byBU, _ := s.queryTop(fmt.Sprintf("SELECT business_unit, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY business_unit ORDER BY sum(%s) DESC LIMIT 10", costCol, costCol))

	c.JSON(http.StatusOK, gin.H{
		"by_service":     byService,
		"by_environment": byEnv,
		"by_region":      byRegion,
		"by_business_unit": byBU,
	})
}

func (s *CostAnalyticsService) queryTop(query string) ([]gin.H, error) {
	rows, err := s.ch.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []gin.H
	for rows.Next() {
		var name string
		var cost float64
		if err := rows.Scan(&name, &cost); err != nil {
			continue
		}
		if name == "" {
			name = "(not set)"
		}
		results = append(results, gin.H{"name": name, "cost": cost})
	}
	return results, nil
}

func (s *CostAnalyticsService) getBudgets(c *gin.Context) {
	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	costCol := "effective_cost"
	if currency == "BRL" {
		costCol = "effective_cost_brl"
	}

	rows, err := s.pg.Query(`SELECT id, name, amount, spent, period, start_date, end_date, alert_threshold, provider, account_id, created_at FROM budgets ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		var startDate, endDate sql.NullTime
		if err := rows.Scan(&b.ID, &b.Name, &b.Amount, &b.Spent, &b.Period, &startDate, &endDate, &b.AlertThreshold, &b.Provider, &b.AccountID, &b.CreatedAt); err != nil {
			log.Printf("[BUDGETS] Scan error: %v", err)
			continue
		}
		if startDate.Valid {
			b.StartDate = startDate.Time
		}
		if endDate.Valid {
			b.EndDate = endDate.Time
		}
		// Compute spent from ClickHouse real data
		startStr := b.StartDate.Format("2006-01-02")
		endStr := b.EndDate.Format("2006-01-02")
		b.Spent = s.computeSpent(b.Provider, b.AccountID, costCol, startStr, endStr)
		b.Remaining = b.Amount - b.Spent
		budgets = append(budgets, b)
	}
	if budgets == nil {
		budgets = []Budget{}
	}
	c.JSON(http.StatusOK, budgets)
}

func (s *CostAnalyticsService) computeSpent(provider, accountID, costCol, startDate, endDate string) float64 {
	query := fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date BETWEEN ? AND ?", costCol)
	args := []interface{}{startDate, endDate}
	if provider != "" && provider != "all" {
		query += " AND provider = ?"
		args = append(args, provider)
	}
	if accountID != "" && accountID != "all" {
		query += " AND billing_account_id = ?"
		args = append(args, accountID)
	}
	var spent *float64
	row := s.ch.QueryRow(context.Background(), query, args...)
	row.Scan(&spent)
	return derefFloat(spent)
}

func (s *CostAnalyticsService) computeBudgetUtilization(costCol string) float64 {
	var budgeted, spent *float64
	pgRow := s.pg.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM budgets`)
	pgRow.Scan(&budgeted)
	chRow := s.ch.QueryRow(context.Background(), fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", costCol))
	chRow.Scan(&spent)
	b := derefFloat(budgeted)
	sv := derefFloat(spent)
	if b > 0 {
		return sv / b
	}
	return 0
}

func (s *CostAnalyticsService) createBudget(c *gin.Context) {
	var b Budget
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := s.pg.Exec(`INSERT INTO budgets (name, amount, period, start_date, end_date, alert_threshold, provider, account_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		b.Name, b.Amount, b.Period, b.StartDate, b.EndDate, b.AlertThreshold, b.Provider, b.AccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

func (s *CostAnalyticsService) updateBudget(c *gin.Context) {
	id := c.Param("id")
	var b Budget
	c.ShouldBindJSON(&b)
	_, err := s.pg.Exec(`UPDATE budgets SET name=$1, amount=$2, alert_threshold=$3 WHERE id=$4`, b.Name, b.Amount, b.AlertThreshold, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (s *CostAnalyticsService) deleteBudget(c *gin.Context) {
	id := c.Param("id")
	_, err := s.pg.Exec(`DELETE FROM budgets WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *CostAnalyticsService) getUsers(c *gin.Context) {
	rows, err := s.pg.Query(`SELECT id, email, name, roles, active, last_login, created_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var users []gin.H
	for rows.Next() {
		var id, email, name string
		var roles []string
		var active bool
		var lastLogin sql.NullTime
		var created time.Time
		if err := rows.Scan(&id, &email, &name, &roles, &active, &lastLogin, &created); err != nil {
			log.Printf("[USERS] Scan error: %v", err)
			continue
		}
		u := gin.H{
			"id":         id,
			"email":      email,
			"name":       name,
			"roles":      roles,
			"active":     active,
			"created_at": created,
		}
		if lastLogin.Valid {
			u["last_login"] = lastLogin.Time
		}
		users = append(users, u)
	}
	if users == nil {
		users = []gin.H{}
	}
	c.JSON(http.StatusOK, users)
}

func (s *CostAnalyticsService) createUser(c *gin.Context) {
	var req struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := s.pg.Exec(`INSERT INTO users (email, name, password_hash, roles) VALUES ($1, $2, $3, $4)`, req.Email, req.Name, req.Password, req.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "created"})
}

func (s *CostAnalyticsService) updateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
		Active   *bool    `json:"active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Password != "" {
		_, err := s.pg.Exec(`UPDATE users SET email=$1, name=$2, password_hash=$3, roles=$4, active=$5 WHERE id=$6`,
			req.Email, req.Name, req.Password, req.Roles, req.Active != nil && *req.Active, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := s.pg.Exec(`UPDATE users SET email=$1, name=$2, roles=$3, active=$4 WHERE id=$5`,
			req.Email, req.Name, req.Roles, req.Active != nil && *req.Active, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (s *CostAnalyticsService) deleteUser(c *gin.Context) {
	id := c.Param("id")
	_, err := s.pg.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *CostAnalyticsService) getAccounts(c *gin.Context) {
	rows, err := s.pg.Query(`SELECT id, provider, account_id, account_name, active FROM cloud_accounts ORDER BY account_name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var accounts []gin.H
	for rows.Next() {
		var id, provider, accountID, accountName string
		var active bool
		rows.Scan(&id, &provider, &accountID, &accountName, &active)
		accounts = append(accounts, gin.H{
			"id":           id,
			"provider":     provider,
			"account_id":   accountID,
			"account_name": accountName,
			"active":       active,
		})
	}
	c.JSON(http.StatusOK, accounts)
}

func (s *CostAnalyticsService) getAudit(c *gin.Context) {
	rows, _ := s.pg.Query(`SELECT id, user_id, action, resource, details, created_at FROM audit ORDER BY created_at DESC LIMIT 100`)
	var audits []gin.H
	for rows.Next() {
		var id, userID, action, resource, details string
		var created time.Time
		rows.Scan(&id, &userID, &action, &resource, &details, &created)
		audits = append(audits, gin.H{"id": id, "user_id": userID, "action": action, "resource": resource, "details": details, "created_at": created})
	}
	rows.Close()
	c.JSON(http.StatusOK, audits)
}

type TruncateDatesRequest struct {
	Provider        string   `json:"provider" binding:"required"`
	AccountID       string   `json:"account_id" binding:"required"`
	Dates           []string `json:"dates" binding:"required"`
}

func (s *CostAnalyticsService) truncateDates(c *gin.Context) {
	var req TruncateDatesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Dates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dates is required"})
		return
	}

	// Validate date format to prevent injection.
	for _, d := range req.Dates {
		if _, err := time.Parse("2006-01-02", d); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date: " + d})
			return
		}
	}

	dateList := "'" + strings.Join(req.Dates, "','") + "'"
	query := fmt.Sprintf(
		"ALTER TABLE finops.costs_raw DELETE WHERE provider = '%s' AND billing_account_id = '%s' AND date IN (%s)",
		req.Provider, req.AccountID, dateList,
	)
	if err := s.ch.Exec(c.Request.Context(), query); err != nil {
		log.Printf("[ADMIN] truncate dates failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "deleted_dates": req.Dates})
}

func healthHandler(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) }
func readyHandler(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ready"}) }
func liveHandler(c *gin.Context)   { c.JSON(http.StatusOK, gin.H{"status": "alive"}) }

func metricsHandler(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4")
	c.String(http.StatusOK, "# Cost Analytics Metrics\ncost_analytics_up 1\n")
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
