package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/time/rate"
)

// Config
var jwtSecret = []byte(getEnv("JWT_SECRET", "finops-enterprise-secret-key"))
var rateLimit = rate.Limit(getEnvFloat("RATE_LIMIT_RPS", 100))
var rateBurst = getEnvInt("RATE_LIMIT_BURST", 200)

// RBAC definitions
var rolePermissions = map[string][]string{
	"admin":      {"*"},
	"analyst":    {"GET:/api/v1/costs", "GET:/api/v1/forecast", "GET:/api/v1/anomalies", "GET:/api/v1/recommendations", "GET:/api/v1/budgets", "POST:/api/v1/budgets", "GET:/api/v1/alerts"},
	"viewer":     {"GET:/api/v1/costs", "GET:/api/v1/dashboard", "GET:/api/v1/budgets"},
}

type Claims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func main() {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())
	r.Use(securityHeaders())
	r.Use(structuredLogger())

	// Health & Observability
	r.GET("/health", healthHandler)
	r.HEAD("/health", healthHandler)
	r.GET("/ready", readyHandler)
	r.HEAD("/ready", readyHandler)
	r.GET("/live", liveHandler)
	r.HEAD("/live", liveHandler)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Auth routes
	r.POST("/api/v1/auth/login", loginHandler)
	r.POST("/api/v1/auth/refresh", refreshHandler)

	// Protected API
	api := r.Group("/api/v1")
	api.Use(jwtMiddleware())
	api.Use(rateLimitMiddleware())
	api.Use(auditMiddleware())
	api.Use(rbacMiddleware())

	// Cost Analytics
	api.GET("/costs", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/trends", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/services", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/applications", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/environments", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/business-units", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/regions", proxyTo("http://cost-analytics:8082"))
	api.GET("/costs/providers", proxyTo("http://cost-analytics:8082"))
	api.GET("/kpis", proxyTo("http://cost-analytics:8082"))
	api.GET("/dashboard/executive", proxyTo("http://cost-analytics:8082"))
	api.GET("/dashboard/operational", proxyTo("http://cost-analytics:8082"))
	api.GET("/anomalies", proxyTo("http://cost-analytics:8082"))
	api.GET("/accounts", proxyTo("http://cost-analytics:8082"))

	// Forecast (real data from cost-analytics)
	api.GET("/forecast", proxyTo("http://cost-analytics:8082"))
	api.GET("/forecast/:period", proxyTo("http://cost-analytics:8082"))
	api.POST("/forecast", proxyTo("http://cost-analytics:8082"))

	// Anomalies (ML service: detect / get by ID)
	api.GET("/anomalies/:id", proxyTo("http://anomaly-detection:8002"))
	api.POST("/anomalies", proxyTo("http://anomaly-detection:8002"))

	// Recommendations (real data from cost-analytics)
	api.GET("/recommendations", proxyTo("http://cost-analytics:8082"))
	api.POST("/recommendations", proxyTo("http://cost-analytics:8082"))
	api.POST("/recommendations/:id/apply", proxyTo("http://cost-analytics:8082"))

	// Budgets
	api.GET("/budgets", proxyTo("http://cost-analytics:8082"))
	api.POST("/budgets", proxyTo("http://cost-analytics:8082"))
	api.PUT("/budgets/:id", proxyTo("http://cost-analytics:8082"))
	api.DELETE("/budgets/:id", proxyTo("http://cost-analytics:8082"))

	// Alerts
	api.GET("/alerts", proxyTo("http://alert-manager:8083"))
	api.POST("/alerts", proxyTo("http://alert-manager:8083"))
	api.PUT("/alerts/:id", proxyTo("http://alert-manager:8083"))
	api.DELETE("/alerts/:id", proxyTo("http://alert-manager:8083"))

	// Ingestion
	api.POST("/ingestion/trigger", proxyTo("http://ingestion-service:8081"))
	api.POST("/ingestion/reprocess", proxyTo("http://ingestion-service:8081"))
	api.GET("/ingestion/status", proxyTo("http://ingestion-service:8081"))
	api.GET("/ingestion/checkpoints", proxyTo("http://ingestion-service:8081"))

	// Admin
	api.GET("/admin/users", proxyTo("http://cost-analytics:8082"))
	api.POST("/admin/users", proxyTo("http://cost-analytics:8082"))
	api.PUT("/admin/users/:id", proxyTo("http://cost-analytics:8082"))
	api.DELETE("/admin/users/:id", proxyTo("http://cost-analytics:8082"))
	api.GET("/admin/audit", proxyTo("http://cost-analytics:8082"))

	port := getEnv("PORT", "8080")
	log.Printf("API Gateway starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// JWT Middleware
func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims := token.Claims.(*Claims)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

// RBAC Middleware
func rbacMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, _ := c.Get("roles")
		roleList, ok := roles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no roles"})
			return
		}
		method := c.Request.Method
		path := c.Request.URL.Path
		allowed := false
		for _, role := range roleList {
			perms, exists := rolePermissions[role]
			if !exists {
				continue
			}
			for _, perm := range perms {
				if perm == "*" || perm == method+":"+path {
					allowed = true
					break
				}
				// Wildcard prefix match
				if strings.HasSuffix(perm, "/*") && strings.HasPrefix(path, strings.TrimSuffix(perm, "/*")) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// Rate Limiting
var limiters = make(map[string]*rate.Limiter)

func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		key, ok := userID.(string)
		if !ok {
			key = c.ClientIP()
		}
		lim, exists := limiters[key]
		if !exists {
			lim = rate.NewLimiter(rateLimit, rateBurst)
			limiters[key] = lim
		}
		if !lim.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// Audit Middleware
func auditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		email, _ := c.Get("email")
		log.Printf("[AUDIT] user=%s email=%s method=%s path=%s ip=%s time=%s",
			userID, email, c.Request.Method, c.Request.URL.Path, c.ClientIP(), time.Now().UTC().Format(time.RFC3339))
		c.Next()
	}
}

// Proxy helper
func proxyTo(target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetURL, err := url.Parse(target)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid proxy target"})
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("[PROXY] error: %v", err)
			w.WriteHeader(http.StatusBadGateway)
		}
		proxy.Director = func(req *http.Request) {
			req.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = c.Request.URL.Path
			req.URL.RawQuery = c.Request.URL.RawQuery
			req.Header = c.Request.Header.Clone()
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

// Auth handlers
func loginHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate against DB (simplified)
	if req.Email == "admin@finops.local" && req.Password == "admin123" {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
			UserID: "1",
			Email:  req.Email,
			Roles:  []string{"admin"},
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			},
		})
		tokenStr, _ := token.SignedString(jwtSecret)
		c.JSON(http.StatusOK, gin.H{"token": tokenStr, "user": gin.H{"id": "1", "email": req.Email, "roles": []string{"admin"}}})
		return
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
}

func refreshHandler(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := jwt.ParseWithClaims(req.Token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	claims := token.Claims.(*Claims)
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Roles:  claims.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})
	tokenStr, _ := newToken.SignedString(jwtSecret)
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

// Health handlers
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "api-gateway", "timestamp": time.Now().UTC()})
}
func readyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
func liveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Security headers
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func structuredLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf(`{"time":"%s","client":"%s","method":"%s","path":"%s","status":%d,"latency":"%s"}`+"\n",
 param.TimeStamp.Format(time.RFC3339),
 param.ClientIP,
 param.Method,
 param.Path,
 param.StatusCode,
 param.Latency.String())
	})
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
func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
