package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	SomeOtherID  int    `json:"some_other_id"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type RequestLog struct {
	Timestamp  string  `json:"timestamp"`
	Method     string  `json:"method"`
	Endpoint   string  `json:"endpoint"`
	StatusCode int     `json:"status_code"`
	DurationMs float64 `json:"duration_ms"`
}

var (
	products = map[int]Product{
		1: {ProductID: 1, SKU: "ABC-123-XYZ", Manufacturer: "Acme Corporation", CategoryID: 10, Weight: 1250, SomeOtherID: 100},
		2: {ProductID: 2, SKU: "DEF-456-UVW", Manufacturer: "Globex Corp", CategoryID: 20, Weight: 500, SomeOtherID: 200},
		3: {ProductID: 3, SKU: "GHI-789-RST", Manufacturer: "Initech", CategoryID: 30, Weight: 750, SomeOtherID: 300},
	}
	mu sync.RWMutex

	// Prometheus metrics
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of non-2xx HTTP responses",
		},
		[]string{"method", "endpoint", "status"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	logger = log.New(os.Stdout, "", 0)
)

func init() {
	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(requestDuration)
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Prometheus metrics
		requestCounter.WithLabelValues(method, endpoint, status).Inc()
		requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())

		if c.Writer.Status() >= 400 {
			errorCounter.WithLabelValues(method, endpoint, status).Inc()
		}

		// Structured JSON log
		logEntry := RequestLog{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			Method:     method,
			Endpoint:   endpoint,
			StatusCode: c.Writer.Status(),
			DurationMs: float64(duration.Microseconds()) / 1000.0,
		}
		logJSON, _ := json.Marshal(logEntry)
		logger.Println(string(logJSON))
	}
}

func main() {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(metricsMiddleware())

	r.GET("/products/:productId", getProduct)
	r.POST("/products/:productId/details", addProductDetails)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	fmt.Println("Product service listening on :8080")
	r.Run(":8080")
}

func getProduct(c *gin.Context) {
	idStr := c.Param("productId")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
			Details: fmt.Sprintf("Invalid product ID: %s", idStr),
		})
		return
	}

	mu.RLock()
	product, exists := products[id]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
			Details: fmt.Sprintf("No product with ID %d", id),
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

func addProductDetails(c *gin.Context) {
	idStr := c.Param("productId")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid product ID in URL",
			Details: fmt.Sprintf("Product ID must be a positive integer, got: %s", idStr),
		})
		return
	}

	var input Product
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	if input.SKU == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Missing required field",
			Details: "sku is required and cannot be empty",
		})
		return
	}
	if len(input.SKU) > 100 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid field value",
			Details: "sku must be at most 100 characters",
		})
		return
	}
	if input.Manufacturer == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Missing required field",
			Details: "manufacturer is required and cannot be empty",
		})
		return
	}
	if len(input.Manufacturer) > 200 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid field value",
			Details: "manufacturer must be at most 200 characters",
		})
		return
	}
	if input.CategoryID < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid field value",
			Details: "category_id must be a positive integer",
		})
		return
	}
	if input.Weight < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid field value",
			Details: "weight must be non-negative",
		})
		return
	}
	if input.SomeOtherID < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid field value",
			Details: "some_other_id must be a positive integer",
		})
		return
	}

	if input.ProductID != 0 && input.ProductID != id {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Product ID mismatch",
			Details: fmt.Sprintf("URL product ID %d does not match body product_id %d", id, input.ProductID),
		})
		return
	}

	mu.Lock()
	_, exists := products[id]
	if !exists {
		mu.Unlock()
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
			Details: fmt.Sprintf("No product with ID %d", id),
		})
		return
	}

	input.ProductID = id
	products[id] = input
	mu.Unlock()

	c.Status(http.StatusNoContent)
}
