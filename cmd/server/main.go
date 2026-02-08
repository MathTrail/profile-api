package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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

	// Setup Gin router with all routes, Dapr handlers, and Swagger
	router := server.NewRouter(container.ProfileController, container.EventHandler, container)

	// Create HTTP server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown: listen for SIGTERM/SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Start server in a goroutine
	go func() {
		log.Printf("Profile Service starting on %s", addr)
		log.Printf("Swagger UI available at http://localhost:%s/swagger/index.html", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Block until signal received
	<-ctx.Done()
	log.Println("Shutdown signal received, draining connections...")

	// Give in-flight requests time to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("Server stopped gracefully")
}
