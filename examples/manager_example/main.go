package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chronnie/governance/manager"
	"github.com/chronnie/governance/models"
)

func main() {
	// Zap logger can be enabled by setting environment variables:
	// export GOVERNANCE_LOG_ENABLED=true
	// export GOVERNANCE_LOG_LEVEL=debug  # or info, warn, error
	// export GOVERNANCE_LOG_FORMAT=json  # or console (default)

	log.Println("Starting governance manager example...")

	// Create manager configuration
	config := &models.ManagerConfig{
		ServerPort:           8080,
		HealthCheckInterval:  30 * time.Second,
		HealthCheckTimeout:   5 * time.Second,
		HealthCheckRetry:     3,
		NotificationInterval: 60 * time.Second,
		NotificationTimeout:  5 * time.Second,
		EventQueueSize:       1000,
	}

	// Create and start manager
	mgr := manager.NewManager(config)
	if err := mgr.Start(); err != nil {
		log.Fatalf("Failed to start manager: %v", err)
	}

	log.Println("Manager started successfully!")
	log.Println("Endpoints:")
	log.Println("  - POST   http://localhost:8080/register")
	log.Println("  - DELETE http://localhost:8080/unregister")
	log.Println("  - GET    http://localhost:8080/services")
	log.Println("  - GET    http://localhost:8080/health")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	log.Println("Shutting down manager...")
	if err := mgr.Stop(); err != nil {
		log.Fatalf("Failed to stop manager: %v", err)
	}

	log.Println("Manager stopped successfully")
}
