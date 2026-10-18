package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
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

var (
	products = map[int]Product{
		1: {ProductID: 1, SKU: "ABC-123-XYZ", Manufacturer: "Acme Corporation", CategoryID: 10, Weight: 1250, SomeOtherID: 100},
		2: {ProductID: 2, SKU: "DEF-456-UVW", Manufacturer: "Globex Corp", CategoryID: 20, Weight: 500, SomeOtherID: 200},
		3: {ProductID: 3, SKU: "GHI-789-RST", Manufacturer: "Initech", CategoryID: 30, Weight: 750, SomeOtherID: 300},
	}
	mu sync.RWMutex
)

func main() {
	r := gin.Default()

	r.GET("/products/:productId", getProduct)
	r.POST("/products/:productId/details", addProductDetails)

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

	// Validate required fields
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

	// Check if URL productId matches body product_id
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
