package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

type AlertManager struct {
	pg         *sql.DB
	kafkaRdr   *kafka.Reader
	httpClient *http.Client
}

type AlertRule struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Condition   string    `json:"condition"`
	Threshold   float64   `json:"threshold"`
	Severity    string    `json:"severity"`
	Channel     string    `json:"channel"`
	Destination string    `json:"destination"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
}

type Alert struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Notification struct {
	Channel     string `json:"channel"`
	Destination string `json:"destination"`
	Subject     string `json:"subject"`
	Body        string `json:"body"`
}

func main() {
	pgConn := getEnv("POSTGRES_DSN", "postgres://finops:finops@postgres:5432/finops?sslmode=disable")
	pg, err := sql.Open("postgres", pgConn)
	if err != nil {
		log.Fatal(err)
	}

	kafkaRdr := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{getEnv("KAFKA_BROKERS", "kafka:9092")},
		Topic:   "anomaly.alerts",
		GroupID: "alert-manager",
	})

	svc := &AlertManager{
		pg:         pg,
		kafkaRdr:   kafkaRdr,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	go svc.consumeAlerts()

	r := gin.Default()
	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/live", liveHandler)

	api := r.Group("/api/v1")
	api.GET("/alerts", svc.getAlerts)
	api.POST("/alerts", svc.createAlertRule)
	api.PUT("/alerts/:id", svc.updateAlertRule)
	api.DELETE("/alerts/:id", svc.deleteAlertRule)
	api.POST("/alerts/:id/resolve", svc.resolveAlert)
	api.GET("/alerts/history", svc.getAlertHistory)

	port := getEnv("PORT", "8083")
	log.Printf("Alert Manager starting on port %s", port)
	r.Run(":" + port)
}

func (s *AlertManager) consumeAlerts() {
	for {
		msg, err := s.kafkaRdr.ReadMessage(nil)
		if err != nil {
			log.Printf("[KAFKA] Read error: %v", err)
			continue
		}
		var alert Alert
		json.Unmarshal(msg.Value, &alert)

		// Store alert
		_, err = s.pg.Exec(`INSERT INTO alerts (rule_id, rule_name, severity, message, value, threshold, status) VALUES ($1, $2, $3, $4, $5, $6, 'firing')`,
			alert.RuleID, alert.RuleName, alert.Severity, alert.Message, alert.Value, alert.Threshold)
		if err != nil {
			log.Printf("[ALERT] DB insert failed: %v", err)
			continue
		}

		// Send notification
		s.sendNotification(alert)
	}
}

func (s *AlertManager) sendNotification(alert Alert) {
	// Get rule configuration
	var rule AlertRule
	row := s.pg.QueryRow(`SELECT channel, destination FROM alert_rules WHERE id = $1`, alert.RuleID)
	row.Scan(&rule.Channel, &rule.Destination)

	notif := Notification{
		Channel:     rule.Channel,
		Destination: rule.Destination,
		Subject:     "FinOps Alert: " + alert.RuleName,
		Body:        alert.Message + "
Value: " + string(rune(int(alert.Value))) + "
Threshold: " + string(rune(int(alert.Threshold))),
	}

	switch rule.Channel {
	case "slack":
		s.sendSlack(notif)
	case "email":
		s.sendEmail(notif)
	case "webhook":
		s.sendWebhook(notif)
	default:
		log.Printf("[ALERT] Unknown channel: %s", rule.Channel)
	}
}

func (s *AlertManager) sendSlack(n Notification) {
	payload := map[string]string{"text": n.Body}
	data, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(n.Destination, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[SLACK] Send failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("[SLACK] Notification sent to %s", n.Destination)
}

func (s *AlertManager) sendEmail(n Notification) {
	log.Printf("[EMAIL] Would send to %s: %s", n.Destination, n.Subject)
}

func (s *AlertManager) sendWebhook(n Notification) {
	payload := map[string]interface{}{
		"subject": n.Subject,
		"body":    n.Body,
		"timestamp": time.Now().UTC(),
	}
	data, _ := json.Marshal(payload)
	resp, err := s.httpClient.Post(n.Destination, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[WEBHOOK] Send failed: %v", err)
		return
	}
	resp.Body.Close()
	log.Printf("[WEBHOOK] Notification sent to %s", n.Destination)
}

func (s *AlertManager) getAlerts(c *gin.Context) {
	status := c.DefaultQuery("status", "firing")
	rows, err := s.pg.Query(`SELECT id, rule_id, rule_name, severity, message, value, threshold, status, created_at, resolved_at FROM alerts WHERE status = $1 ORDER BY created_at DESC LIMIT 100`, status)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var alerts []Alert
	for rows.Next() {
		var a Alert
		rows.Scan(&a.ID, &a.RuleID, &a.RuleName, &a.Severity, &a.Message, &a.Value, &a.Threshold, &a.Status, &a.CreatedAt, &a.ResolvedAt)
		alerts = append(alerts, a)
	}
	c.JSON(200, alerts)
}

func (s *AlertManager) createAlertRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	_, err := s.pg.Exec(`INSERT INTO alert_rules (name, description, condition, threshold, severity, channel, destination, enabled) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		rule.Name, rule.Description, rule.Condition, rule.Threshold, rule.Severity, rule.Channel, rule.Destination, rule.Enabled)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, rule)
}

func (s *AlertManager) updateAlertRule(c *gin.Context) {
	id := c.Param("id")
	var rule AlertRule
	c.ShouldBindJSON(&rule)
	_, err := s.pg.Exec(`UPDATE alert_rules SET name=$1, description=$2, condition=$3, threshold=$4, severity=$5, channel=$6, destination=$7, enabled=$8 WHERE id=$9`,
		rule.Name, rule.Description, rule.Condition, rule.Threshold, rule.Severity, rule.Channel, rule.Destination, rule.Enabled, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "updated"})
}

func (s *AlertManager) deleteAlertRule(c *gin.Context) {
	id := c.Param("id")
	_, err := s.pg.Exec(`DELETE FROM alert_rules WHERE id=$1`, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
}

func (s *AlertManager) resolveAlert(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	_, err := s.pg.Exec(`UPDATE alerts SET status='resolved', resolved_at=$1 WHERE id=$2`, now, id)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "resolved"})
}

func (s *AlertManager) getAlertHistory(c *gin.Context) {
	rows, err := s.pg.Query(`SELECT id, rule_id, rule_name, severity, message, value, threshold, status, created_at, resolved_at FROM alerts ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var alerts []Alert
	for rows.Next() {
		var a Alert
		rows.Scan(&a.ID, &a.RuleID, &a.RuleName, &a.Severity, &a.Message, &a.Value, &a.Threshold, &a.Status, &a.CreatedAt, &a.ResolvedAt)
		alerts = append(alerts, a)
	}
	c.JSON(200, alerts)
}

func healthHandler(c *gin.Context) { c.JSON(200, gin.H{"status": "healthy"}) }
func readyHandler(c *gin.Context)  { c.JSON(200, gin.H{"status": "ready"}) }
func liveHandler(c *gin.Context)   { c.JSON(200, gin.H{"status": "alive"}) }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}
