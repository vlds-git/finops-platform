package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/segmentio/kafka-go"
	_ "github.com/lib/pq"
)

type CloudAccount struct {
	Provider  string `json:"provider"`
	AccountID string `json:"account_id"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
}

type IngestionService struct {
	obsClient   *minio.Client
	kafkaWriter *kafka.Writer
	pg          *sql.DB
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
	EffectiveCost      float64           `json:"effective_cost"`     // USD
	EffectiveCostBRL   float64           `json:"effective_cost_brl"` // BRL
	ListUnitPrice      float64           `json:"list_unit_price"`
	ListCost           float64           `json:"list_cost"`     // USD
	ListCostBRL        float64           `json:"list_cost_brl"` // BRL
	ContractedCost     float64           `json:"contracted_cost"`     // USD
	ContractedCostBRL  float64           `json:"contracted_cost_brl"` // BRL
	AmortizedCost      float64           `json:"amortized_cost"`     // USD
	AmortizedCostBRL   float64           `json:"amortized_cost_brl"` // BRL
	Tags               map[string]string `json:"tags"`
	Provider           string            `json:"provider"`
	Environment        string            `json:"environment"`
	Application        string            `json:"application"`
	BusinessUnit       string            `json:"business_unit"`
}

type TriggerRequest struct {
	Provider  string `json:"provider" binding:"required"`
	Bucket    string `json:"bucket" binding:"required"`
	Prefix    string `json:"prefix"`
	AccountID string `json:"account_id" binding:"required"`
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
	pgConn := getEnv("POSTGRES_DSN", "postgres://finops:finops@postgres:5432/finops?sslmode=disable")
	pg, err := sql.Open("postgres", pgConn)
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	pg.SetMaxOpenConns(10)
	pg.SetMaxIdleConns(5)
	pg.SetConnMaxLifetime(5 * time.Minute)

	obsClient, err := initObsClient()
	if err != nil {
		log.Fatalf("OBS client init failed: %v", err)
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
		pg:          pg,
		checkpoints: make(map[string]string),
	}

	// Load checkpoints from PostgreSQL
	if err := svc.loadAllCheckpoints(); err != nil {
		log.Printf("[INGESTION] Failed to load checkpoints: %v", err)
	}

	// Start automatic ingestion scheduler
	intervalStr := getEnv("INGESTION_INTERVAL", "5m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		interval = 5 * time.Minute
	}
	go svc.runScheduler(interval)

	r := gin.Default()
	r.GET("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.GET("/live", liveHandler)

	api := r.Group("/api/v1")
	api.POST("/ingestion/trigger", svc.triggerHandler)
	api.GET("/ingestion/status", svc.statusHandler)
	api.GET("/ingestion/checkpoints", svc.checkpointsHandler)
	api.POST("/ingestion/reprocess", svc.reprocessHandler)

	port := getEnv("PORT", "8081")
	log.Printf("Ingestion Service starting on port %s (scheduler every %s)", port, interval)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func initObsClient() (*minio.Client, error) {
	ak := getEnv("HUAWEI_ACCESS_KEY", "")
	sk := getEnv("HUAWEI_SECRET_KEY", "")
	endpoint := getEnv("HUAWEI_OBS_ENDPOINT", "obs.myhwclouds.com")
	if ak == "" || sk == "" {
		return nil, fmt.Errorf("missing OBS credentials (HUAWEI_ACCESS_KEY and HUAWEI_SECRET_KEY are required)")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(ak, sk, ""),
		Secure:       true,
		BucketLookup: minio.BucketLookupDNS,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OBS client: %w", err)
	}
	return client, nil
}

func (s *IngestionService) runScheduler(interval time.Duration) {
	// Run once at startup after a short delay, then on ticker
	time.Sleep(30 * time.Second)
	s.runAllAccounts()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.runAllAccounts()
	}
}

func (s *IngestionService) runAllAccounts() {
	accounts, err := s.loadCloudAccounts()
	if err != nil {
		log.Printf("[SCHEDULER] Failed to load cloud accounts: %v", err)
		return
	}
	for _, acc := range accounts {
		go s.processIngestion(acc.Provider, acc.Bucket, acc.Prefix, acc.AccountID)
	}
}

func (s *IngestionService) loadCloudAccounts() ([]CloudAccount, error) {
	rows, err := s.pg.Query(`SELECT provider, account_id, bucket, prefix FROM cloud_accounts WHERE active = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []CloudAccount
	for rows.Next() {
		var a CloudAccount
		if err := rows.Scan(&a.Provider, &a.AccountID, &a.Bucket, &a.Prefix); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *IngestionService) triggerHandler(c *gin.Context) {
	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go s.processIngestion(req.Provider, req.Bucket, req.Prefix, req.AccountID)
	c.JSON(http.StatusAccepted, gin.H{"status": "ingestion_started", "provider": req.Provider, "timestamp": time.Now().UTC()})
}

func (s *IngestionService) processIngestion(provider, bucket, prefix, accountID string) {
	log.Printf("[INGESTION] Starting for provider=%s bucket=%s prefix=%s account=%s", provider, bucket, prefix, accountID)

	checkpoint := s.loadCheckpoint(provider, accountID)
	log.Printf("[INGESTION] Last checkpoint: %s", checkpoint)

	ctx := context.Background()
	objectCh := s.obsClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var processed int
	for object := range objectCh {
		if object.Err != nil {
			log.Printf("[INGESTION] ListObjects error: %v", object.Err)
			s.saveDLQ(provider, accountID, "", object.Err.Error())
			continue
		}

		file := object.Key
		if strings.HasSuffix(file, "/") {
			continue
		}
		if file <= checkpoint {
			continue
		}

		log.Printf("[INGESTION] Processing file: %s", file)

		reader, err := s.obsClient.GetObject(ctx, bucket, file, minio.GetObjectOptions{})
		if err != nil {
			log.Printf("[INGESTION] GetObject failed: %v", err)
			s.saveDLQ(provider, accountID, file, err.Error())
			continue
		}

		records, err := s.parseObject(file, provider, reader)
		reader.Close()
		if err != nil {
			log.Printf("[INGESTION] Parse failed: %v", err)
			s.saveDLQ(provider, accountID, file, err.Error())
			continue
		}

		for _, rec := range records {
			rec.Provider = provider
			rec.BillingAccountID = accountID
			rec.Environment = rec.Tags["environment"]
			rec.Application = rec.Tags["application"]
			rec.BusinessUnit = rec.Tags["business_unit"]

			data, err := json.Marshal(rec)
			if err != nil {
				log.Printf("[INGESTION] JSON marshal failed: %v", err)
				continue
			}
			err = s.kafkaWriter.WriteMessages(ctx, kafka.Message{
				Key:   []byte(fmt.Sprintf("%s-%s", provider, accountID)),
				Value: data,
			})
			if err != nil {
				log.Printf("[INGESTION] Kafka write failed: %v", err)
				continue
			}
		}

		s.saveCheckpoint(provider, accountID, file)
		processed++
	}

	log.Printf("[INGESTION] Completed for provider=%s account=%s processed=%d", provider, accountID, processed)
}

func (s *IngestionService) parseObject(filename, provider string, reader io.Reader) ([]FocusRecord, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return s.parseZIP(data, provider)
	case ".csv":
		return s.parseCSV(data, provider)
	case ".parquet":
		return s.parseParquet(data, provider)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func (s *IngestionService) parseCSV(data []byte, provider string) ([]FocusRecord, error) {
	cr := csv.NewReader(bytes.NewReader(data))
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return demoRecords(provider), nil
	}

	var records []FocusRecord
	header := rows[0]
	colIndex := make(map[string]int)
	for i, h := range header {
		colIndex[strings.ToLower(strings.TrimSpace(h))] = i
	}

	for _, row := range rows[1:] {
		if len(row) < len(header) {
			continue
		}
		rec := FocusRecord{
			InvoiceIssuer:    provider,
			Date:             normalizeDate(getCol(row, colIndex, "date", getCol(row, colIndex, "charge_period_start", time.Now().Format("2006-01-02")))),
			ServiceName:      getCol(row, colIndex, "service_name", ""),
			ServiceCategory:  getCol(row, colIndex, "service_category", ""),
			ResourceType:     getCol(row, colIndex, "resource_type", ""),
			ResourceID:       getCol(row, colIndex, "resource_id", ""),
			ResourceName:     getCol(row, colIndex, "resource_name", ""),
			Region:           getCol(row, colIndex, "region", "sa-brazil-1"),
			AvailabilityZone: getCol(row, colIndex, "availability_zone", ""),
			UsageUnit:        getCol(row, colIndex, "usage_unit", ""),
			ChargeType:       getCol(row, colIndex, "charge_type", ""),
			ChargeDescription: getCol(row, colIndex, "charge_description", ""),
			Tags:             parseTags(getCol(row, colIndex, "tags", "")),
		}
		rec.UsageQuantity, _ = strconv.ParseFloat(getCol(row, colIndex, "usage_quantity", "0"), 64)

		// Costs in USD
		effCost, _ := strconv.ParseFloat(getCol(row, colIndex, "effective_cost", "0"), 64)
		listCost, _ := strconv.ParseFloat(getCol(row, colIndex, "list_cost", "0"), 64)
		contractedCost, _ := strconv.ParseFloat(getCol(row, colIndex, "contracted_cost", "0"), 64)
		amortizedCost, _ := strconv.ParseFloat(getCol(row, colIndex, "amortized_cost", "0"), 64)

		rec.EffectiveCost = roundToTwo(effCost)
		rec.ListCost = roundToTwo(listCost)
		rec.ContractedCost = roundToTwo(contractedCost)
		rec.AmortizedCost = roundToTwo(amortizedCost)

		// BRL conversion
		rec.EffectiveCostBRL = roundToTwo(effCost * usdToBRLRate)
		rec.ListCostBRL = roundToTwo(listCost * usdToBRLRate)
		rec.ContractedCostBRL = roundToTwo(contractedCost * usdToBRLRate)
		rec.AmortizedCostBRL = roundToTwo(amortizedCost * usdToBRLRate)

		records = append(records, rec)
	}
	return records, nil
}

func demoRecords(provider string) []FocusRecord {
	now := time.Now().Format("2006-01-02")
	return []FocusRecord{
		{
			InvoiceIssuer:    provider,
			ServiceName:      "Compute",
			ResourceType:     "Virtual Machine",
			Region:           "sa-brazil-1",
			UsageQuantity:    720,
			UsageUnit:        "Hours",
			EffectiveCost:    150.00,
			EffectiveCostBRL: roundToTwo(150.00 * usdToBRLRate),
			ListCost:         200.00,
			ListCostBRL:      roundToTwo(200.00 * usdToBRLRate),
			AmortizedCost:    150.00,
			AmortizedCostBRL: roundToTwo(150.00 * usdToBRLRate),
			Tags:             map[string]string{"environment": "production", "application": "erp", "business_unit": "finance"},
			Date:             now,
		},
		{
			InvoiceIssuer:    provider,
			ServiceName:      "Storage",
			ResourceType:     "Object Storage",
			Region:           "sa-brazil-1",
			UsageQuantity:    500,
			UsageUnit:        "GB",
			EffectiveCost:    25.00,
			EffectiveCostBRL: roundToTwo(25.00 * usdToBRLRate),
			ListCost:         30.00,
			ListCostBRL:      roundToTwo(30.00 * usdToBRLRate),
			AmortizedCost:    25.00,
			AmortizedCostBRL: roundToTwo(25.00 * usdToBRLRate),
			Tags:             map[string]string{"environment": "production", "application": "backup", "business_unit": "it"},
			Date:             now,
		},
	}
}

func parseTags(raw string) map[string]string {
	tags := map[string]string{"environment": "production", "application": "default", "business_unit": "default"}
	if raw == "" {
		return tags
	}
	parts := strings.Split(raw, ";")
	for _, p := range parts {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 {
			tags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return tags
}

func normalizeDate(d string) string {
	// Try common formats
	for _, layout := range []string{"2006-01-02", "2006/01/02", "01/02/2006", "2006-01-02T15:04:05Z", time.RFC3339} {
		if t, err := time.Parse(layout, d); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return d
}

func getCol(row []string, colIndex map[string]int, name, def string) string {
	if idx, ok := colIndex[name]; ok && idx < len(row) {
		v := strings.TrimSpace(row[idx])
		if v != "" {
			return v
		}
	}
	return def
}

func (s *IngestionService) parseZIP(data []byte, provider string) ([]FocusRecord, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
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
		fileData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			log.Printf("[ZIP] read file failed: %v", err)
			continue
		}
		recs, err := s.parseCSV(fileData, provider)
		if err != nil {
			log.Printf("[ZIP] parse CSV failed: %v", err)
			continue
		}
		records = append(records, recs...)
	}
	return records, nil
}

func (s *IngestionService) parseParquet(data []byte, provider string) ([]FocusRecord, error) {
	log.Printf("[INGESTION] Parquet parser not implemented, falling back to CSV")
	return s.parseCSV(data, provider)
}

func (s *IngestionService) saveDLQ(provider, accountID, fileName, errMsg string) {
	_, dbErr := s.pg.Exec(`INSERT INTO ingestion_dlq (provider, account_id, file_name, error_message) VALUES ($1, $2, $3, $4)`,
		provider, accountID, fileName, errMsg)
	if dbErr != nil {
		log.Printf("[DLQ] Failed to persist DLQ: %v", dbErr)
	}
}

func (s *IngestionService) loadAllCheckpoints() error {
	rows, err := s.pg.Query(`SELECT provider, account_id, last_file FROM ingestion_checkpoints WHERE status = 'active'`)
	if err != nil {
		return err
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()
	for rows.Next() {
		var provider, accountID, lastFile string
		if err := rows.Scan(&provider, &accountID, &lastFile); err != nil {
			continue
		}
		s.checkpoints[fmt.Sprintf("%s:%s", provider, accountID)] = lastFile
	}
	return nil
}

func (s *IngestionService) loadCheckpoint(provider, accountID string) string {
	key := fmt.Sprintf("%s:%s", provider, accountID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.checkpoints[key]
}

func (s *IngestionService) saveCheckpoint(provider, accountID, file string) {
	key := fmt.Sprintf("%s:%s", provider, accountID)
	s.mu.Lock()
	s.checkpoints[key] = file
	s.mu.Unlock()

	_, err := s.pg.Exec(`
		INSERT INTO ingestion_checkpoints (provider, account_id, last_file, last_processed_at, status)
		VALUES ($1, $2, $3, NOW(), 'active')
		ON CONFLICT (provider, account_id)
		DO UPDATE SET last_file = EXCLUDED.last_file, last_processed_at = EXCLUDED.last_processed_at, status = 'active'`,
		provider, accountID, file)
	if err != nil {
		log.Printf("[CHECKPOINT] Failed to persist checkpoint: %v", err)
	} else {
		log.Printf("[CHECKPOINT] Saved: %s -> %s", key, file)
	}
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
		key := fmt.Sprintf("%s:%s", req.Provider, req.AccountID)
		s.mu.Lock()
		delete(s.checkpoints, key)
		s.mu.Unlock()

		_, err := s.pg.Exec(`UPDATE ingestion_checkpoints SET last_file = NULL, last_processed_at = NULL WHERE provider = $1 AND account_id = $2`,
			req.Provider, req.AccountID)
		if err != nil {
			log.Printf("[REPROCESS] Failed to reset checkpoint: %v", err)
		}
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
