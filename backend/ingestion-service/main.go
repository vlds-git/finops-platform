package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/segmentio/kafka-go"
)

type IngestionService struct {
	obsClient   *obs.ObsClient
	kafkaWriter *kafka.Writer
	checkpoints map[string]string
	mu          sync.RWMutex
}

type FocusRecord struct {
	InvoiceIssuer      string            `json:"invoice_issuer"`
	InvoiceID          string            `json:"invoice_id"`
	BillingPeriodStart string            `json:"billing_period_start"`
	BillingPeriodEnd   string            `json:"billing_period_end"`
	ChargePeriodStart  string            `json:"charge_period_start"`
	ChargePeriodEnd    string            `json:"charge_period_end"`
	Date               string            `json:"date"`
	BillingAccountID   string            `json:"billing_account_id"`
	BillingAccountName string            `json:"billing_account_name"`
	BillingProfileID   string            `json:"billing_profile_id"`
	BillingProfileName string            `json:"billing_profile_name"`
	ChargeType         string            `json:"charge_type"`
	ChargeSubcategory  string            `json:"charge_subcategory"`
	ChargeDescription  string            `json:"charge_description"`
	ServiceName        string            `json:"service_name"`
	ServiceCategory    string            `json:"service_category"`
	ResourceType       string            `json:"resource_type"`
	ResourceID         string            `json:"resource_id"`
	ResourceName       string            `json:"resource_name"`
	Region             string            `json:"region"`
	AvailabilityZone   string            `json:"availability_zone"`
	UsageUnit          string            `json:"usage_unit"`
	UsageQuantity      float64           `json:"usage_quantity"`
	EffectiveCost      float64           `json:"effective_cost"`
	ListUnitPrice      float64           `json:"list_unit_price"`
	ListCost           float64           `json:"list_cost"`
	ContractedCost     float64           `json:"contracted_cost"`
	AmortizedCost      float64           `json:"amortized_cost"`
	Tags               map[string]string `json:"tags"`
	Provider           string            `json:"provider"`
	Environment        string            `json:"environment"`
	Application        string            `json:"application"`
	BusinessUnit       string            `json:"business_unit"`
}

type Checkpoint struct {
	Provider   string    `json:"provider"`
	AccountID  string    `json:"account_id"`
	LastFile   string    `json:"last_file"`
	LastOffset int64     `json:"last_offset"`
	UpdatedAt  time.Time `json:"updated_at"`
}

var usdToBRLRate float64

func init() {
	rateStr := getEnv("USD_TO_BRL_RATE", "5.15")
	if r, err := strconv.ParseFloat(rateStr, 64); err == nil {
		usdToBRLRate = r
	} else {
		usdToBRLRate = 5.15
	}
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
	defer kafkaWriter.Close()

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
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
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

	checkpoint := s.loadCheckpoint(provider, accountID)
	log.Printf("[INGESTION] Last checkpoint: %s", checkpoint.LastFile)

	files := []string{"focus_export_2024_01.csv", "focus_export_2024_02.csv", "focus_export_2024_03.parquet", "focus_export_2024_04.zip"}
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

			data, err := json.Marshal(rec)
			if err != nil {
				log.Printf("[INGESTION] JSON marshal failed: %v", err)
				s.sendToDLQ(rec, err)
				continue
			}
			err = s.kafkaWriter.WriteMessages(context.Background(), kafka.Message{
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
	var records []FocusRecord
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".zip":
		records = s.parseZIP(filename, provider)
	case ".csv":
		records = s.parseCSV(filename, provider)
	case ".parquet":
		records = s.parseParquet(filename, provider)
	}
	return records
}

func (s *IngestionService) parseCSV(filename, provider string) []FocusRecord {
	return []FocusRecord{
		{
			InvoiceIssuer: provider,
			ServiceName:   "Compute",
			ResourceType:  "Virtual Machine",
			Region:        "sa-brazil-1",
			UsageQuantity: 720,
			UsageUnit:     "Hours",
			EffectiveCost: roundToTwo(150.00 * usdToBRLRate),
			ListCost:      roundToTwo(200.00 * usdToBRLRate),
			AmortizedCost: roundToTwo(150.00 * usdToBRLRate),
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
			EffectiveCost: roundToTwo(25.00 * usdToBRLRate),
			ListCost:      roundToTwo(30.00 * usdToBRLRate),
			AmortizedCost: roundToTwo(25.00 * usdToBRLRate),
			Tags:          map[string]string{"environment": "production", "application": "backup", "business_unit": "it"},
			Date:          "2024-01-15",
		},
	}
}

func (s *IngestionService) parseZIP(filename, provider string) []FocusRecord {
	// Simulated ZIP processing: create a minimal in-memory ZIP containing one CSV,
	// then extract and parse the first data row.
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	fc, err := zw.Create("focus_export.csv")
	if err != nil {
		log.Printf("[ZIP] create entry failed: %v", err)
		zw.Close()
		return nil
	}
	csvWriter := csv.NewWriter(fc)
	_ = csvWriter.Write([]string{"service_name", "resource_type", "region", "usage_quantity", "usage_unit", "effective_cost", "date"})
	_ = csvWriter.Write([]string{"Compute", "Virtual Machine", "sa-brazil-1", "720", "Hours", "120.00", "2024-01-15"})
	csvWriter.Flush()
	zw.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		log.Printf("[ZIP] open reader failed: %v", err)
		return nil
	}

	var records []FocusRecord
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			log.Printf("[ZIP] open file failed: %v", err)
			continue
		}
		defer rc.Close()

		cr := csv.NewReader(rc)
		rows, err := cr.ReadAll()
		if err != nil {
			log.Printf("[ZIP] csv read failed: %v", err)
			continue
		}
		for i, row := range rows {
			if i == 0 {
				continue
			}
			if len(row) < 7 {
				continue
			}
			usage, _ := strconv.ParseFloat(row[3], 64)
			cost, _ := strconv.ParseFloat(row[5], 64)
			records = append(records, FocusRecord{
				InvoiceIssuer: provider,
				ServiceName:   row[0],
				ResourceType:  row[1],
				Region:        row[2],
				UsageQuantity: usage,
				UsageUnit:     row[4],
				EffectiveCost: roundToTwo(cost * usdToBRLRate),
				ListCost:      roundToTwo(cost * usdToBRLRate),
				AmortizedCost: roundToTwo(cost * usdToBRLRate),
				Tags:          map[string]string{"environment": "production", "application": "erp", "business_unit": "finance"},
				Date:          row[6],
			})
		}
	}
	return records
}

func (s *IngestionService) parseParquet(filename, provider string) []FocusRecord {
	return s.parseCSV(filename, provider)
}

func (s *IngestionService) sendToDLQ(rec FocusRecord, err error) {
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
		"usd_to_brl":  usdToBRLRate,
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func roundToTwo(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
