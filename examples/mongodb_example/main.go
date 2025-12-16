package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chronnie/governance/manager"
	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/storage/mongodb"
)

func main() {
	// MongoDB configuration
	mongoConfig := mongodb.Config{
		URI:            "mongodb://localhost:27017",
		Database:       "governance",
		ConnectTimeout: 10 * time.Second,
		// Optional connection pool settings
		MaxPoolSize: 100,
		MinPoolSize: 10,
	}

	// Create MongoDB database store
	db, err := mongodb.NewDatabaseStore(mongoConfig)
	if err != nil {
		log.Fatalf("Failed to create MongoDB database: %v", err)
	}
	log.Println("MongoDB database initialized successfully")

	// Manager configuration
	managerConfig := &models.ManagerConfig{
		ServerPort:           8080,
		HealthCheckInterval:  30 * time.Second,
		NotificationInterval: 60 * time.Second,
		HealthCheckTimeout:   5 * time.Second,
		NotificationTimeout:  5 * time.Second,
		HealthCheckRetry:     3,
		EventQueueSize:       1000,
	}

	// Create manager with MongoDB database persistence (cache + database)
	mgr := manager.NewManagerWithDatabase(managerConfig, db)

	// Start manager
	if err := mgr.Start(); err != nil {
		log.Fatalf("Failed to start manager: %v", err)
	}

	log.Println("Governance manager with MongoDB storage started")
	log.Println("REST API available at http://localhost:8080")
	log.Println("Endpoints:")
	log.Println("  POST   /register   - Register a service")
	log.Println("  POST   /unregister - Unregister a service")
	log.Println("  GET    /services   - List all services")
	log.Println("  GET    /health     - Health check")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	if err := mgr.Stop(); err != nil {
		log.Printf("Error stopping manager: %v", err)
	}

	log.Println("Manager stopped")
}
