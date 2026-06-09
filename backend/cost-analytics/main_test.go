package main

import (
	"net/http"
	"net/http/httptest"
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

func TestGetKPIs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &CostAnalyticsService{}
	r := gin.New()
	r.GET("/kpis", svc.getKPIs)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/kpis", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Cost Efficiency")
	assert.Contains(t, w.Body.String(), "Budget Utilization")
}

func TestGetExecutiveDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &CostAnalyticsService{}
	r := gin.New()
	r.GET("/dashboard/executive", svc.getExecutiveDashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/executive", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_cost")
	assert.Contains(t, w.Body.String(), "forecast_30d")
	assert.Contains(t, w.Body.String(), "potential_savings")
}

func TestGetOperationalDashboard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &CostAnalyticsService{}
	r := gin.New()
	r.GET("/dashboard/operational", svc.getOperationalDashboard)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/dashboard/operational", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "by_service")
	assert.Contains(t, w.Body.String(), "by_environment")
}
