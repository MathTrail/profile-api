package main

import (
	"fmt"
	"log"

	"github.com/mathtrail/mathtrail-profile/internal/app"
	"github.com/mathtrail/mathtrail-profile/internal/config"
	"github.com/mathtrail/mathtrail-profile/internal/server"

	// Import generated Swagger docs (will be created by swag init)
	_ "github.com/mathtrail/mathtrail-profile/docs"
)

// @title MathTrail Profile Service API
// @version 1.0
// @description Profile service for the MathTrail ecosystem. Manages user profiles, skills, and progress.

// @host localhost:8080
// @BasePath /api/v1

// @schemes http
func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize DI container
	container := app.NewContainer(cfg)
	defer container.Close()

	// Setup Gin router with all routes and Swagger
	router := server.NewRouter(container.ProfileController)

	// TODO: Register Dapr pub/sub endpoints (Stage 4)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Profile Service starting on %s", addr)
	log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.ServerPort)

	if err := router.Run(addr); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
