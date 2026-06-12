package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	ServiceCount  int     `json:"service_count"`
	ResourceCount int     `json:"resource_count"`
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

	kafkaRdr := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{getEnv("KAFKA_BROKERS", "kafka:9092")},
		Topic:   "cost.raw",
		GroupID: "cost-analytics",
	})

	svc := &CostAnalyticsService{pg: pg, ch: chConn, kafkaRdr: kafkaRdr}
	go svc.consumeKafka()
	go svc.runDailyAggregator()

	r := gin.Default()
	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/live", liveHandler)

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

	api.GET("/budgets", svc.getBudgets)
	api.POST("/budgets", svc.createBudget)
	api.PUT("/budgets/:id", svc.updateBudget)
	api.DELETE("/budgets/:id", svc.deleteBudget)

	api.GET("/admin/users", svc.getUsers)
	api.POST("/admin/users", svc.createUser)
	api.GET("/admin/audit", svc.getAudit)

	port := getEnv("PORT", "8082")
	log.Printf("Cost Analytics starting on port %s", port)
	r.Run(":" + port)
}

func (s *CostAnalyticsService) consumeKafka() {
	ctx := context.Background()
	for {
		msg, err := s.kafkaRdr.ReadMessage(ctx)
		if err != nil {
			log.Printf("[KAFKA] Read error: %v", err)
			continue
		}

		var rec FocusRecord
		if err := json.Unmarshal(msg.Value, &rec); err != nil {
			log.Printf("[KAFKA] Unmarshal error: %v", err)
			continue
		}

		if err := s.insertRawCost(ctx, rec); err != nil {
			log.Printf("[KAFKA] ClickHouse insert error: %v", err)
			continue
		}
		log.Printf("[KAFKA] Persisted record: %s/%s/%s", rec.Provider, rec.ServiceName, rec.Date)
	}
}

func (s *CostAnalyticsService) insertRawCost(ctx context.Context, rec FocusRecord) error {
	if rec.Tags == nil {
		rec.Tags = map[string]string{}
	}
	query := `
		INSERT INTO costs_raw (
			provider, billing_account_id, service_name, resource_type, resource_id, region,
			usage_quantity, usage_unit, effective_cost, effective_cost_brl, list_cost, list_cost_brl,
			contracted_cost, contracted_cost_brl, amortized_cost, amortized_cost_brl,
			date, environment, application, business_unit, tags
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	return s.ch.AsyncInsert(ctx, query, false,
		rec.Provider, rec.BillingAccountID, rec.ServiceName, rec.ResourceType, rec.ResourceID, rec.Region,
		rec.UsageQuantity, rec.UsageUnit, rec.EffectiveCost, rec.EffectiveCostBRL, rec.ListCost, rec.ListCostBRL,
		rec.ContractedCost, rec.ContractedCostBRL, rec.AmortizedCost, rec.AmortizedCostBRL,
		rec.Date, rec.Environment, rec.Application, rec.BusinessUnit, rec.Tags,
	)
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

func (s *CostAnalyticsService) getCosts(c *gin.Context) {
	start := c.Query("start_date")
	end := c.Query("end_date")
	provider := c.Query("provider")
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")

	if start == "" || end == "" {
		start = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
		end = time.Now().Format("2006-01-02")
	}

	query := fmt.Sprintf(`SELECT sum(%s), sum(amortized_cost), sum(list_cost), uniqExact(service_name), uniqExact(resource_id) FROM costs_raw WHERE date BETWEEN ? AND ?`, costCol)
	args := []interface{}{start, end}
	if provider != "" {
		query += " AND provider = ?"
		args = append(args, provider)
	}

	row := s.ch.QueryRow(context.Background(), query, args...)
	var summary CostSummary
	if err := row.Scan(&summary.TotalCost, &summary.AmortizedCost, &summary.ListCost, &summary.ServiceCount, &summary.ResourceCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (s *CostAnalyticsService) getTrends(c *gin.Context) {
	period := c.DefaultQuery("period", "30")
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT date, sum(%s) as cost, sum(usage_quantity) as usage FROM costs_raw WHERE date >= today() - interval ? day GROUP BY date ORDER BY date`, costCol)
	rows, err := s.ch.Query(context.Background(), query, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []gin.H
	for rows.Next() {
		var date string
		var cost, usage float64
		rows.Scan(&date, &cost, &usage)
		results = append(results, gin.H{"date": date, "cost": cost, "usage": usage})
	}
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByService(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT service_name, sum(%s) as cost, sum(usage_quantity) as usage FROM costs_raw WHERE date >= today() - 30 GROUP BY service_name ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var svc string
		var cost, usage float64
		rows.Scan(&svc, &cost, &usage)
		results = append(results, gin.H{"service": svc, "cost": cost, "usage": usage})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByApplication(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT application, sum(%s) as cost FROM costs_raw WHERE date >= today() - 30 GROUP BY application ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var app string
		var cost float64
		rows.Scan(&app, &cost)
		results = append(results, gin.H{"application": app, "cost": cost})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByEnvironment(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT environment, sum(%s) as cost FROM costs_raw WHERE date >= today() - 30 GROUP BY environment ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var env string
		var cost float64
		rows.Scan(&env, &cost)
		results = append(results, gin.H{"environment": env, "cost": cost})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByBusinessUnit(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT business_unit, sum(%s) as cost FROM costs_raw WHERE date >= today() - 30 GROUP BY business_unit ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var bu string
		var cost float64
		rows.Scan(&bu, &cost)
		results = append(results, gin.H{"business_unit": bu, "cost": cost})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByRegion(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT region, sum(%s) as cost FROM costs_raw WHERE date >= today() - 30 GROUP BY region ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var region string
		var cost float64
		rows.Scan(&region, &cost)
		results = append(results, gin.H{"region": region, "cost": cost})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getByProvider(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	query := fmt.Sprintf(`SELECT provider, sum(%s) as cost FROM costs_raw WHERE date >= today() - 30 GROUP BY provider ORDER BY cost DESC`, costCol)
	rows, _ := s.ch.Query(context.Background(), query)
	var results []gin.H
	for rows.Next() {
		var provider string
		var cost float64
		rows.Scan(&provider, &cost)
		results = append(results, gin.H{"provider": provider, "cost": cost})
	}
	rows.Close()
	c.JSON(http.StatusOK, results)
}

func (s *CostAnalyticsService) getKPIs(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	var total, forecast float64
	row := s.ch.QueryRow(context.Background(), fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", costCol))
	row.Scan(&total)
	row = s.ch.QueryRow(context.Background(), fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 60 AND date < today() - 30", costCol))
	var previous float64
	row.Scan(&previous)

	trend := 0.0
	if previous > 0 {
		trend = (total - previous) / previous
	}
	forecast = total * (1 + trend)

	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	kpis := []KPIData{
		{Name: "Total Cost (30d)", Value: total, Unit: currency, Trend: trend, Status: statusForTrend(trend)},
		{Name: "Forecast (next 30d)", Value: forecast, Unit: currency, Trend: trend, Status: statusForTrend(trend)},
		{Name: "Cost Efficiency", Value: 0.85, Unit: "ratio", Trend: 0.03, Target: 0.90, Status: "warning"},
		{Name: "Budget Utilization", Value: 0.72, Unit: "ratio", Trend: 0.05, Target: 0.80, Status: "good"},
	}
	c.JSON(http.StatusOK, kpis)
}

func statusForTrend(trend float64) string {
	if trend > 0.05 {
		return "warning"
	}
	return "good"
}

func (s *CostAnalyticsService) getExecutiveDashboard(c *gin.Context) {
	costCol := currencyColumn(c, "effective_cost", "effective_cost_brl")
	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))

	var totalCost float64
	row := s.ch.QueryRow(context.Background(), fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", costCol))
	row.Scan(&totalCost)

	topServices, _ := s.queryTop(fmt.Sprintf("SELECT service_name, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY service_name ORDER BY sum(%s) DESC LIMIT 5", costCol, costCol))
	topApps, _ := s.queryTop(fmt.Sprintf("SELECT application, sum(%s) FROM costs_raw WHERE date >= today() - 30 GROUP BY application ORDER BY sum(%s) DESC LIMIT 5", costCol, costCol))

	c.JSON(http.StatusOK, gin.H{
		"total_cost":        totalCost,
		"forecast_30d":      totalCost * 1.05,
		"potential_savings": totalCost * 0.15,
		"top_services":      topServices,
		"top_applications":  topApps,
		"currency":          currency,
	})
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

	rows, err := s.pg.Query(`SELECT id, name, amount, spent, period, start_date, end_date, alert_threshold, provider, account_id, created_at FROM budgets`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var budgets []Budget
	for rows.Next() {
		var b Budget
		rows.Scan(&b.ID, &b.Name, &b.Amount, &b.Spent, &b.Period, &b.StartDate, &b.EndDate, &b.AlertThreshold, &b.Provider, &b.AccountID, &b.CreatedAt)
		// Compute spent from ClickHouse real data
		b.Spent = s.computeSpent(b.Provider, b.AccountID, costCol)
		b.Remaining = b.Amount - b.Spent
		budgets = append(budgets, b)
	}
	c.JSON(http.StatusOK, budgets)
}

func (s *CostAnalyticsService) computeSpent(provider, accountID, costCol string) float64 {
	query := fmt.Sprintf("SELECT sum(%s) FROM costs_raw WHERE date >= today() - 30", costCol)
	args := []interface{}{}
	if provider != "" && provider != "all" {
		query += " AND provider = ?"
		args = append(args, provider)
	}
	if accountID != "" && accountID != "all" {
		query += " AND billing_account_id = ?"
		args = append(args, accountID)
	}
	var spent float64
	row := s.ch.QueryRow(context.Background(), query, args...)
	row.Scan(&spent)
	return spent
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
	rows, _ := s.pg.Query(`SELECT id, email, name, roles, created_at FROM users`)
	var users []gin.H
	for rows.Next() {
		var id, email, name string
		var roles []string
		var created time.Time
		rows.Scan(&id, &email, &name, &roles, &created)
		users = append(users, gin.H{"id": id, "email": email, "name": name, "roles": roles, "created_at": created})
	}
	rows.Close()
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

func healthHandler(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) }
func readyHandler(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ready"}) }
func liveHandler(c *gin.Context)   { c.JSON(http.StatusOK, gin.H{"status": "alive"}) }

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
