package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
)

type IngestionService struct {
	obsClient   *obs.ObsClient
	kafkaWriter *kafka.Writer
	checkpoints map[string]string
	mu          sync.RWMutex
}

type FocusRecord struct {
	InvoiceIssuer    string  `json:"invoice_issuer"`
	InvoiceID        string  `json:"invoice_id"`
	BillingPeriodStart string `json:"billing_period_start"`
	BillingPeriodEnd   string `json:"billing_period_end"`
	ChargePeriodStart  string `json:"charge_period_start"`
	ChargePeriodEnd    string `json:"charge_period_end"`
	Date             string  `json:"date"`
	BillingAccountID string  `json:"billing_account_id"`
	BillingAccountName string `json:"billing_account_name"`
	BillingProfileID string  `json:"billing_profile_id"`
	BillingProfileName string `json:"billing_profile_name"`
	ChargeType       string  `json:"charge_type"`
	ChargeSubcategory string `json:"charge_subcategory"`
	ChargeDescription string  `json:"charge_description"`
	ServiceName      string  `json:"service_name"`
	ServiceCategory  string  `json:"service_category"`
	ResourceType     string  `json:"resource_type"`
	ResourceID       string  `json:"resource_id"`
	ResourceName     string  `json:"resource_name"`
	Region           string  `json:"region"`
	AvailabilityZone string  `json:"availability_zone"`
	UsageUnit        string  `json:"usage_unit"`
	UsageQuantity    float64 `json:"usage_quantity"`
	EffectiveCost    float64 `json:"effective_cost"`
	ListUnitPrice    float64 `json:"list_unit_price"`
	ListCost         float64 `json:"list_cost"`
	ContractedCost   float64 `json:"contracted_cost"`
	AmortizedCost    float64 `json:"amortized_cost"`
	Tags             map[string]string `json:"tags"`
	Provider         string  `json:"provider"`
	Environment      string  `json:"environment"`
	Application      string  `json:"application"`
	BusinessUnit     string  `json:"business_unit"`
}

type Checkpoint struct {
	Provider   string    `json:"provider"`
	AccountID  string    `json:"account_id"`
	LastFile   string    `json:"last_file"`
	LastOffset int64     `json:"last_offset"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func main() {
	obsClient, err := initObsClient()
	if err != nil {
		log.Printf("OBS client init failed (non-fatal for demo): %v", err)
	}

	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(getEnv("KAFKA_BROKERS", "kafka:9092")),
		Topic:    "cost.raw",
		Balancer: &kafka.LeastBytes{},
	}

	svc := &IngestionService{
		obsClient:   obsClient,
		kafkaWriter: kafkaWriter,
		checkpoints: make(map[string]string),
	}

	r := gin.Default()
	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/live", liveHandler)
	r.POST("/ingestion/trigger", svc.triggerHandler)
	r.GET("/ingestion/status", svc.statusHandler)
	r.GET("/ingestion/checkpoints", svc.checkpointsHandler)
	r.POST("/ingestion/reprocess", svc.reprocessHandler)

	port := getEnv("PORT", "8081")
	log.Printf("Ingestion Service starting on port %s", port)
	r.Run(":" + port)
}

func initObsClient() (*obs.ObsClient, error) {
	ak := getEnv("HUAWEI_ACCESS_KEY", "")
	sk := getEnv("HUAWEI_SECRET_KEY", "")
	endpoint := getEnv("HUAWEI_OBS_ENDPOINT", "obs.myhwclouds.com")
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("missing OBS credentials")
	}
	return obs.New(ak, sk, endpoint)
}

func (s *IngestionService) triggerHandler(c *gin.Context) {
	var req struct {
		Provider  string `json:"provider" binding:"required"`
		Bucket    string `json:"bucket" binding:"required"`
		Prefix    string `json:"prefix"`
		AccountID string `json:"account_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go s.processIngestion(req.Provider, req.Bucket, req.Prefix, req.AccountID)
	c.JSON(http.StatusAccepted, gin.H{"status": "ingestion_started", "provider": req.Provider, "timestamp": time.Now().UTC()})
}

func (s *IngestionService) processIngestion(provider, bucket, prefix, accountID string) {
	log.Printf("[INGESTION] Starting for provider=%s bucket=%s prefix=%s", provider, bucket, prefix)

	// In real implementation: list objects from OBS, filter by checkpoint
	// For demo: simulate processing
	checkpoint := s.loadCheckpoint(provider, accountID)
	log.Printf("[INGESTION] Last checkpoint: %s", checkpoint.LastFile)

	// Simulate file discovery
	files := []string{"focus_export_2024_01.csv", "focus_export_2024_02.csv", "focus_export_2024_03.parquet"}
	for _, file := range files {
		if file <= checkpoint.LastFile {
			continue
		}
		log.Printf("[INGESTION] Processing file: %s", file)

		records := s.parseFile(file, provider)
		for _, rec := range records {
			rec.Provider = provider
			rec.BillingAccountID = accountID
			rec.Environment = rec.Tags["environment"]
			rec.Application = rec.Tags["application"]
			rec.BusinessUnit = rec.Tags["business_unit"]

			data, _ := json.Marshal(rec)
			err := s.kafkaWriter.WriteMessages(context.Background(), kafka.Message{
				Key:   []byte(fmt.Sprintf("%s-%s", provider, accountID)),
				Value: data,
			})
			if err != nil {
				log.Printf("[INGESTION] Kafka write failed: %v", err)
				s.sendToDLQ(rec, err)
				continue
			}
		}

		s.saveCheckpoint(provider, accountID, file, 0)
	}
	log.Printf("[INGESTION] Completed for provider=%s", provider)
}

func (s *IngestionService) parseFile(filename, provider string) []FocusRecord {
	// Simulated parser - in production handles CSV/Parquet with proper schema
	var records []FocusRecord
	ext := strings.ToLower(filepath.Ext(filename))

	if ext == ".csv" {
		records = s.parseCSV(filename, provider)
	} else if ext == ".parquet" {
		records = s.parseParquet(filename, provider)
	}
	return records
}

func (s *IngestionService) parseCSV(filename, provider string) []FocusRecord {
	// Simulated CSV parsing with realistic data
	return []FocusRecord{
		{
			InvoiceIssuer: provider,
			ServiceName:   "Compute",
			ResourceType:  "Virtual Machine",
			Region:        "sa-brazil-1",
			UsageQuantity: 720,
			UsageUnit:     "Hours",
			EffectiveCost: 150.00,
			ListCost:      200.00,
			AmortizedCost: 150.00,
			Tags:          map[string]string{"environment": "production", "application": "erp", "business_unit": "finance"},
			Date:          "2024-01-15",
		},
		{
			InvoiceIssuer: provider,
			ServiceName:   "Storage",
			ResourceType:  "Object Storage",
			Region:        "sa-brazil-1",
			UsageQuantity: 500,
			UsageUnit:     "GB",
			EffectiveCost: 25.00,
			ListCost:      30.00,
			AmortizedCost: 25.00,
			Tags:          map[string]string{"environment": "production", "application": "backup", "business_unit": "it"},
			Date:          "2024-01-15",
		},
	}
}

func (s *IngestionService) parseParquet(filename, provider string) []FocusRecord {
	// Parquet parsing would use parquet-go library
	return s.parseCSV(filename, provider) // Simulated
}

func (s *IngestionService) sendToDLQ(rec FocusRecord, err error) {
	// Send to dead letter queue for retry
	log.Printf("[DLQ] Record failed: %v, error: %v", rec.ResourceID, err)
}

func (s *IngestionService) loadCheckpoint(provider, accountID string) Checkpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := fmt.Sprintf("%s:%s", provider, accountID)
	lastFile := s.checkpoints[key]
	return Checkpoint{Provider: provider, AccountID: accountID, LastFile: lastFile, UpdatedAt: time.Now()}
}

func (s *IngestionService) saveCheckpoint(provider, accountID, file string, offset int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%s", provider, accountID)
	s.checkpoints[key] = file
	log.Printf("[CHECKPOINT] Saved: %s -> %s", key, file)
}

func (s *IngestionService) statusHandler(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"status":      "running",
		"checkpoints": len(s.checkpoints),
		"timestamp":   time.Now().UTC(),
	})
}

func (s *IngestionService) checkpointsHandler(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c.JSON(http.StatusOK, s.checkpoints)
}

func (s *IngestionService) reprocessHandler(c *gin.Context) {
	var req struct {
		Provider  string `json:"provider"`
		AccountID string `json:"account_id"`
		FromFile  string `json:"from_file"`
	}
	c.ShouldBindJSON(&req)
	// Reset checkpoint and reprocess
	if req.Provider != "" && req.AccountID != "" {
		s.mu.Lock()
		key := fmt.Sprintf("%s:%s", req.Provider, req.AccountID)
		delete(s.checkpoints, key)
		s.mu.Unlock()
	}
	c.JSON(http.StatusOK, gin.H{"status": "checkpoint_reset", "message": "Reprocessamento agendado"})
}

func healthHandler(c *gin.Context) { c.JSON(200, gin.H{"status": "healthy"}) }
func readyHandler(c *gin.Context)  { c.JSON(200, gin.H{"status": "ready"}) }
func liveHandler(c *gin.Context)   { c.JSON(200, gin.H{"status": "alive"}) }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}
