package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHealthHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", healthHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")
}

func TestStatusHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &IngestionService{
		checkpoints: make(map[string]string),
	}
	r := gin.New()
	r.GET("/ingestion/status", svc.statusHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ingestion/status", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "running")
}

func TestTriggerHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &IngestionService{
		checkpoints: make(map[string]string),
	}
	r := gin.New()
	r.POST("/ingestion/trigger", svc.triggerHandler)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ingestion/trigger", strings.NewReader(`{"provider":"huawei","bucket":"test","account_id":"hw-001"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "ingestion_started")
}

func TestCheckpointOperations(t *testing.T) {
	svc := &IngestionService{
		checkpoints: make(map[string]string),
	}

	// Test save
	svc.saveCheckpoint("huawei", "hw-001", "file1.csv", 0)
	cp := svc.loadCheckpoint("huawei", "hw-001")
	assert.Equal(t, "file1.csv", cp.LastFile)

	// Test reprocess
	svc.mu.Lock()
	delete(svc.checkpoints, "huawei:hw-001")
	svc.mu.Unlock()
	cp = svc.loadCheckpoint("huawei", "hw-001")
	assert.Equal(t, "", cp.LastFile)
}

func TestParseCSV(t *testing.T) {
	svc := &IngestionService{}
	records := svc.parseCSV([]byte("test.csv"), "huawei")
	assert.NotEmpty(t, records)
	assert.Equal(t, "huawei", records[0].InvoiceIssuer)
	assert.Equal(t, "Compute", records[0].ServiceName)
}
