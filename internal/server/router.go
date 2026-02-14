package server

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/MathTrail/profile-api/internal/dapr"
	"github.com/MathTrail/profile-api/internal/profile"
)

// ReadinessChecker is called by the readiness probe to verify downstream dependencies.
type ReadinessChecker interface {
	Ready(ctx context.Context) error
}

// NewRouter creates a Gin engine with all routes, middleware, and Swagger UI.
// It replaces gin.Default() with zap-based request logging and panic recovery.
func NewRouter(profileController *profile.Controller, eventHandler *dapr.EventHandler, readiness ReadinessChecker, logger *zap.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Middleware: request ID → zap request logger → panic recovery
	router.Use(RequestID())
	router.Use(ZapLogger(logger.Named("http")))
	router.Use(ZapRecovery(logger.Named("http")))

	// Health probes (mathtrail-service-lib contract)
	router.GET("/health/startup", startupCheck)
	router.GET("/health/liveness", livenessCheck)
	router.GET("/health/ready", readinessCheck(readiness))

	// Legacy health endpoint (kept for backward compatibility)
	router.GET("/health", livenessCheck)

	// API v1 routes
	api := router.Group("/api/v1")
	profileController.RegisterRoutes(api)

	// Dapr pub/sub subscription and event handler routes
	eventHandler.RegisterRoutes(router)

	// Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	return router
}

// startupCheck returns 200 once the process is running.
// The startup probe gives the service time to warm up.
func startupCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

// livenessCheck returns 200 if the process is alive (not deadlocked).
func livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// readinessCheck verifies the service can handle traffic (DB, Redis reachable).
func readinessCheck(checker ReadinessChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := checker.Ready(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
