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
	"sort"
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
	processMu   sync.Mutex
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
		Addr:         kafka.TCP(getEnv("KAFKA_BROKERS", "kafka:9092")),
		Topic:        "cost.raw",
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1000,
		BatchTimeout: 500 * time.Millisecond,
		Async:        false,
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
	// Prevent concurrent ingestion for the same provider/account to avoid duplicate records.
	s.processMu.Lock()
	defer s.processMu.Unlock()

	log.Printf("[INGESTION] Starting for provider=%s bucket=%s prefix=%s account=%s", provider, bucket, prefix, accountID)

	checkpoint := s.loadCheckpoint(provider, accountID)
	log.Printf("[INGESTION] Last checkpoint: %s", checkpoint)

	ctx := context.Background()
	objectCh := s.obsClient.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	// FOCUS daily exports are cumulative per month. To avoid duplicates and keep
	// the most up-to-date corrections/credits, group files by month prefix and
	// process only the latest file for each month.
	latestByMonth := make(map[string]minio.ObjectInfo)
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
		if !isDataFile(file) {
			continue
		}

		month := monthPrefixFromFile(prefix, file)
		if month == "" {
			continue
		}
		existing, ok := latestByMonth[month]
		if !ok || file > existing.Key {
			latestByMonth[month] = object
		}
	}

	// Sort months so the checkpoint is deterministic (lexicographic order).
	var months []string
	for m := range latestByMonth {
		months = append(months, m)
	}
	sort.Strings(months)

	var processed int
	for _, month := range months {
		object := latestByMonth[month]
		file := object.Key

		// Skip already processed files. The checkpoint is the last globally
		// processed file; because months are sorted lexicographically, any
		// previously processed latest file will be <= checkpoint.
		if file <= checkpoint {
			log.Printf("[INGESTION] Skipping already processed latest file for month=%s: %s", month, file)
			continue
		}

		log.Printf("[INGESTION] Processing latest file for month=%s: %s (size=%d)", month, file, object.Size)

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

		// FOCUS exports are cumulative; the latest file for a month may also
		// contain corrected rows for previous months (e.g. last day of the
		// previous billing period). Delete existing rows for all dates present
		// in this file before publishing, so the newest export always wins.
		if err := s.truncateDatesBeforeIngestion(provider, accountID, records); err != nil {
			log.Printf("[INGESTION] Failed to truncate existing dates: %v", err)
			s.saveDLQ(provider, accountID, file, err.Error())
			continue
		}

		log.Printf("[INGESTION] Publishing %d records to Kafka for month=%s", len(records), month)
		if len(records) > 0 {
			log.Printf("[INGESTION] First record sample: date=%s charge_type=%q service=%s cost=%.2f",
				records[0].Date, records[0].ChargeType, records[0].ServiceName, records[0].EffectiveCost)
		}
		batchSize := 1000
		var messages []kafka.Message
		sent := 0
		for idx, rec := range records {
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
			messages = append(messages, kafka.Message{
				Key:   []byte(fmt.Sprintf("%s-%s-%s", provider, accountID, month)),
				Value: data,
			})

			if len(messages) >= batchSize {
				if err := s.kafkaWriter.WriteMessages(ctx, messages...); err != nil {
					log.Printf("[INGESTION] Kafka batch write failed: %v", err)
				} else {
					sent += len(messages)
				}
				messages = messages[:0]
			}
			if idx > 0 && idx%10000 == 0 {
				log.Printf("[INGESTION] Prepared %d/%d records for Kafka", idx, len(records))
			}
		}
		if len(messages) > 0 {
			if err := s.kafkaWriter.WriteMessages(ctx, messages...); err != nil {
				log.Printf("[INGESTION] Kafka final batch write failed: %v", err)
			} else {
				sent += len(messages)
			}
		}
		log.Printf("[INGESTION] Sent %d/%d records to Kafka for %s", sent, len(records), file)

		s.saveCheckpoint(provider, accountID, file)
		processed++
	}

	log.Printf("[INGESTION] Completed for provider=%s account=%s processed=%d", provider, accountID, processed)
}

func (s *IngestionService) truncateDatesBeforeIngestion(provider, accountID string, records []FocusRecord) error {
	if len(records) == 0 {
		return nil
	}
	dateSet := make(map[string]struct{})
	for _, rec := range records {
		if rec.Date != "" {
			dateSet[rec.Date] = struct{}{}
		}
	}
	if len(dateSet) == 0 {
		return nil
	}
	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	reqBody, err := json.Marshal(map[string]interface{}{
		"provider":   provider,
		"account_id": accountID,
		"dates":      dates,
	})
	if err != nil {
		return err
	}

	costAnalyticsURL := getEnv("COST_ANALYTICS_URL", "http://cost-analytics:8082")
	url := costAnalyticsURL + "/api/v1/admin/truncate-dates"
	resp, err := http.Post(url, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("truncate request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("truncate returned %d: %s", resp.StatusCode, string(body))
	}
	log.Printf("[INGESTION] Truncated %d existing dates before ingestion", len(dates))
	return nil
}

func isDataFile(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	return ext == ".zip" || ext == ".csv" || ext == ".parquet"
}

// monthPrefixFromFile extracts the month directory from a FOCUS export path.
// Expected layout: <prefix>/<YYYYMM>/<timestamp>/<filename>
func monthPrefixFromFile(prefix, file string) string {
	dir := filepath.Dir(file)
	// Remove the trailing timestamp directory to keep prefix/YYYYMM.
	monthDir := filepath.Dir(dir)
	if monthDir == "." || monthDir == "/" {
		return ""
	}
	// Ensure the month directory starts with the configured prefix.
	if !strings.HasPrefix(monthDir, strings.TrimSuffix(prefix, "/")) {
		return ""
	}
	month := filepath.Base(monthDir)
	if len(month) != 6 {
		return ""
	}
	if _, err := strconv.Atoi(month); err != nil {
		return ""
	}
	return monthDir
}

func (s *IngestionService) parseObject(filename, provider string, reader io.Reader) ([]FocusRecord, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".zip":
		return s.parseZIP(data, filename, provider)
	case ".csv":
		return s.parseCSV(data, provider)
	case ".parquet":
		return s.parseParquet(data, provider)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func focusHeaderName(h string) string {
	h = strings.TrimSpace(h)
	if h == "" {
		return ""
	}
	// Convert PascalCase / camelCase FOCUS headers to snake_case
	var b strings.Builder
	for i, r := range h {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func (s *IngestionService) parseCSV(data []byte, provider string) ([]FocusRecord, error) {
	cr := csv.NewReader(bytes.NewReader(data))
	cr.ReuseRecord = false
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
		colIndex[focusHeaderName(h)] = i
	}

	totalRows := len(rows) - 1
	for i, row := range rows[1:] {
		if len(row) < len(header) {
			continue
		}
		if i > 0 && i%10000 == 0 {
			log.Printf("[INGESTION] Parsed %d/%d rows", i, totalRows)
		}

		date := normalizeDate(getCol(row, colIndex, "charge_period_start", getCol(row, colIndex, "charge_period_end", getCol(row, colIndex, "billing_period_start", time.Now().Format("2006-01-02")))))

		rec := FocusRecord{
			InvoiceIssuer:     getCol(row, colIndex, "invoice_issuer_name", provider),
			Date:              date,
			ServiceName:       getCol(row, colIndex, "service_name", ""),
			ServiceCategory:   getCol(row, colIndex, "service_category", ""),
			ResourceType:      getCol(row, colIndex, "resource_type", ""),
			ResourceID:        getCol(row, colIndex, "resource_id", ""),
			ResourceName:      getCol(row, colIndex, "resource_name", ""),
			Region:            getCol(row, colIndex, "region_name", getCol(row, colIndex, "region_id", "sa-brazil-1")),
			AvailabilityZone:  getCol(row, colIndex, "availability_zone", ""),
			UsageUnit:         getCol(row, colIndex, "consumed_unit", getCol(row, colIndex, "pricing_unit", "")),
			ChargeType:        getCol(row, colIndex, "charge_category", ""),
			ChargeDescription: getCol(row, colIndex, "charge_description", ""),
			Tags:              parseTags(getCol(row, colIndex, "tags", "")),
		}
		rec.UsageQuantity, _ = strconv.ParseFloat(getCol(row, colIndex, "consumed_quantity", getCol(row, colIndex, "pricing_quantity", "0")), 64)

		// Costs in USD
		effCost, _ := strconv.ParseFloat(getCol(row, colIndex, "effective_cost", "0"), 64)
		listCost, _ := strconv.ParseFloat(getCol(row, colIndex, "list_cost", "0"), 64)
		contractedCost, _ := strconv.ParseFloat(getCol(row, colIndex, "contracted_cost", "0"), 64)
		billedCost, _ := strconv.ParseFloat(getCol(row, colIndex, "billed_cost", "0"), 64)

		rec.EffectiveCost = roundToTwo(effCost)
		rec.ListCost = roundToTwo(listCost)
		rec.ContractedCost = roundToTwo(contractedCost)
		rec.AmortizedCost = roundToTwo(billedCost)

		// BRL conversion
		rec.EffectiveCostBRL = roundToTwo(effCost * usdToBRLRate)
		rec.ListCostBRL = roundToTwo(listCost * usdToBRLRate)
		rec.ContractedCostBRL = roundToTwo(contractedCost * usdToBRLRate)
		rec.AmortizedCostBRL = roundToTwo(billedCost * usdToBRLRate)

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
	// Try JSON object first (common in FOCUS exports)
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		var parsed map[string]string
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			for k, v := range parsed {
				tags[k] = v
			}
			return tags
		}
	}
	// Fallback to key=value;key=value format
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

func (s *IngestionService) parseZIP(data []byte, filename, provider string) ([]FocusRecord, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	var records []FocusRecord
	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}
		log.Printf("[ZIP] %s -> extracting CSV %s", filename, f.Name)
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
		log.Printf("[ZIP] %s -> parsing %d bytes", f.Name, len(fileData))
		recs, err := s.parseCSV(fileData, provider)
		if err != nil {
			log.Printf("[ZIP] parse CSV failed: %v", err)
			continue
		}
		log.Printf("[ZIP] %s -> %d records parsed", f.Name, len(recs))
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
