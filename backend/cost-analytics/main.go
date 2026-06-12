package main

import (
	"context"
	"database/sql"
		"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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
	for {
		msg, err := s.kafkaRdr.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[KAFKA] Read error: %v", err)
			continue
		}
		log.Printf("[KAFKA] Message received: %s", string(msg.Key))
	}
}

func (s *CostAnalyticsService) getCosts(c *gin.Context) {
	start := c.Query("start_date")
	end := c.Query("end_date")
	provider := c.Query("provider")

	query := `SELECT sum(effective_cost), sum(amortized_cost), sum(list_cost), uniqExact(service_name), uniqExact(resource_id) FROM costs WHERE date BETWEEN ? AND ?`
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
	query := `SELECT date, sum(effective_cost) as cost, sum(usage_quantity) as usage FROM costs WHERE date >= today() - interval ? day GROUP BY date ORDER BY date`
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
	query := `SELECT service_name, sum(effective_cost) as cost, sum(usage_quantity) as usage FROM costs WHERE date >= today() - 30 GROUP BY service_name ORDER BY cost DESC`
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
	query := `SELECT application, sum(effective_cost) as cost FROM costs WHERE date >= today() - 30 GROUP BY application ORDER BY cost DESC`
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
	query := `SELECT environment, sum(effective_cost) as cost FROM costs WHERE date >= today() - 30 GROUP BY environment ORDER BY cost DESC`
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
	query := `SELECT business_unit, sum(effective_cost) as cost FROM costs WHERE date >= today() - 30 GROUP BY business_unit ORDER BY cost DESC`
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
	query := `SELECT region, sum(effective_cost) as cost FROM costs WHERE date >= today() - 30 GROUP BY region ORDER BY cost DESC`
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
	query := `SELECT provider, sum(effective_cost) as cost FROM costs WHERE date >= today() - 30 GROUP BY provider ORDER BY cost DESC`
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
	kpis := []KPIData{
		{Name: "Cost Efficiency", Value: 0.85, Unit: "ratio", Trend: 0.03, Target: 0.90, Status: "warning"},
		{Name: "Budget Utilization", Value: 0.72, Unit: "ratio", Trend: 0.05, Target: 0.80, Status: "good"},
		{Name: "Cost per Application", Value: 12500, Unit: "USD", Trend: -0.02, Status: "good"},
		{Name: "Cost per Environment", Value: 45000, Unit: "USD", Trend: 0.01, Status: "good"},
		{Name: "Cost Trend (30d)", Value: 185000, Unit: "USD", Trend: 0.08, Status: "warning"},
		{Name: "Savings Opportunities", Value: 32000, Unit: "USD", Trend: -0.15, Status: "good"},
	}
	c.JSON(http.StatusOK, kpis)
}

func (s *CostAnalyticsService) getExecutiveDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"total_cost": 185000.00,
		"forecast_30d": 195000.00,
		"potential_savings": 32000.00,
		"top_services": []gin.H{
			{"name": "Compute", "cost": 85000},
			{"name": "Storage", "cost": 35000},
			{"name": "Network", "cost": 25000},
		},
		"top_applications": []gin.H{
			{"name": "ERP", "cost": 45000},
			{"name": "CRM", "cost": 32000},
			{"name": "Data Lake", "cost": 28000},
		},
		"kpis": []gin.H{
			{"name": "Cost Efficiency", "value": 0.85},
			{"name": "Budget Health", "value": 0.92},
		},
	})
}

func (s *CostAnalyticsService) getOperationalDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"by_service":     []gin.H{{"name": "Compute", "cost": 85000}, {"name": "Storage", "cost": 35000}},
		"by_project":     []gin.H{{"name": "Project Alpha", "cost": 62000}, {"name": "Project Beta", "cost": 48000}},
		"by_environment": []gin.H{{"name": "Production", "cost": 120000}, {"name": "Staging", "cost": 35000}, {"name": "Development", "cost": 30000}},
		"by_region":      []gin.H{{"name": "sa-brazil-1", "cost": 95000}, {"name": "east-us-1", "cost": 90000}},
		"by_tags":        []gin.H{{"key": "team", "value": "platform", "cost": 75000}},
	})
}

func (s *CostAnalyticsService) getBudgets(c *gin.Context) {
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
		b.Remaining = b.Amount - b.Spent
		budgets = append(budgets, b)
	}
	c.JSON(http.StatusOK, budgets)
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
		var id, email, name, roles string
		var created time.Time
		rows.Scan(&id, &email, &name, &roles, &created)
		users = append(users, gin.H{"id": id, "email": email, "name": name, "roles": roles, "created_at": created})
	}
	rows.Close()
	c.JSON(http.StatusOK, users)
}

func (s *CostAnalyticsService) createUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		Roles    string `json:"roles"`
	}
	c.ShouldBindJSON(&req)
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
