package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/mathtrail/mathtrail-profile/internal/dapr"
	"github.com/mathtrail/mathtrail-profile/internal/profile"
)

// NewRouter creates a Gin engine with all routes, middleware, and Swagger UI.
func NewRouter(profileController *profile.Controller, eventHandler *dapr.EventHandler) *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", healthCheck)

	// API v1 routes
	api := router.Group("/api/v1")
	profileController.RegisterRoutes(api)

	// Dapr pub/sub subscription and event handler routes
	eventHandler.RegisterRoutes(router)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}

// healthCheck godoc
// @Summary Health check
// @Description Returns service health status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string "Service is healthy"
// @Router /health [get]
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
