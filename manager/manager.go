package manager

import (
	"context"
	"fmt"
	"net/http"
	"time"

	eventqueue "github.com/chronnie/go-event-queue"
	"github.com/chronnie/governance/internal/api"
	"github.com/chronnie/governance/internal/notifier"
	"github.com/chronnie/governance/internal/registry"
	"github.com/chronnie/governance/internal/scheduler"
	"github.com/chronnie/governance/internal/worker"
	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/pkg/logger"
	"github.com/chronnie/governance/storage"
	"go.uber.org/zap"
)

// Manager is the main governance manager component
type Manager struct {
	config *models.ManagerConfig

	// Core components
	dualStore     *storage.DualStore // Always uses in-memory cache + optional database
	registry      *registry.Registry
	eventQueue    eventqueue.IEventQueue
	notifier      *notifier.Notifier
	healthChecker *notifier.HealthChecker
	eventWorker   *worker.EventWorker
	queueContext  context.Context
	queueCancel   context.CancelFunc

	// Schedulers
	healthCheckScheduler *scheduler.HealthCheckScheduler
	reconcileScheduler   *scheduler.ReconcileScheduler

	// HTTP server
	httpServer *http.Server

	// Lifecycle
	stopChan chan struct{}
}

// NewManager creates a new governance manager with in-memory cache only (no database persistence)
func NewManager(config *models.ManagerConfig) *Manager {
	return NewManagerWithDatabase(config, nil)
}

// NewManagerWithDatabase creates a new governance manager with optional database persistence.
// The manager always uses in-memory cache for performance.
// If db is not nil, all changes are also persisted to the database asynchronously.
func NewManagerWithDatabase(config *models.ManagerConfig, db storage.DatabaseStore) *Manager {
	if config == nil {
		config = models.DefaultConfig()
	}

	// Create dual-layer storage (always has cache, database is optional)
	dualStore := storage.NewDualStore(db)

	// Create registry with dual store
	reg := registry.NewRegistry(dualStore)

	// Create event queue with Sequential mode for FIFO processing
	queueConfig := eventqueue.EventQueueConfig{
		BufferSize:     config.EventQueueSize,
		ProcessingMode: eventqueue.Sequential, // Sequential for FIFO event processing
	}
	eventQueue := eventqueue.NewEventQueue(queueConfig)

	// Create notifier
	notif := notifier.NewNotifier(config.NotificationTimeout)

	// Create health checker
	healthCheck := notifier.NewHealthChecker(config.HealthCheckTimeout, config.HealthCheckRetry)

	// Create event worker and register handlers
	eventWorker := worker.NewEventWorker(reg, notif, healthCheck, dualStore)
	eventWorker.RegisterHandlers(eventQueue)

	// Create schedulers
	healthCheckScheduler := scheduler.NewHealthCheckScheduler(reg, eventQueue, config.HealthCheckInterval)
	reconcileScheduler := scheduler.NewReconcileScheduler(eventQueue, config.NotificationInterval)

	// Create HTTP handler
	handler := api.NewHandler(reg, eventQueue)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/register", handler.RegisterHandler)
	mux.HandleFunc("/unregister", handler.UnregisterHandler)
	mux.HandleFunc("/services", handler.ServicesHandler)
	mux.HandleFunc("/health", handler.HealthHandler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ServerPort),
		Handler: mux,
	}

	// Create context for queue
	queueCtx, queueCancel := context.WithCancel(context.Background())

	return &Manager{
		config:               config,
		dualStore:            dualStore,
		registry:             reg,
		eventQueue:           eventQueue,
		notifier:             notif,
		healthChecker:        healthCheck,
		eventWorker:          eventWorker,
		healthCheckScheduler: healthCheckScheduler,
		reconcileScheduler:   reconcileScheduler,
		httpServer:           httpServer,
		stopChan:             make(chan struct{}),
		queueContext:         queueCtx,
		queueCancel:          queueCancel,
	}
}

// Start starts the governance manager
func (m *Manager) Start() error {
	logger.Info("Starting governance manager")

	// Start event queue
	go func() {
		if err := m.eventQueue.Start(m.queueContext); err != nil {
			logger.Error("Event queue error", zap.Error(err))
		}
	}()

	// Start schedulers
	go m.healthCheckScheduler.Start()
	go m.reconcileScheduler.Start()

	// Start HTTP server
	go func() {
		logger.Info("HTTP server starting", zap.Int("port", m.config.ServerPort))
		if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()

	logger.Info("Governance manager started successfully",
		zap.Duration("health_check_interval", m.config.HealthCheckInterval),
		zap.Duration("notification_interval", m.config.NotificationInterval),
	)

	return nil
}

// Stop gracefully stops the governance manager
func (m *Manager) Stop() error {
	logger.Info("Stopping governance manager")

	// Stop schedulers
	m.healthCheckScheduler.Stop()
	m.reconcileScheduler.Stop()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Stop event queue
	if err := m.eventQueue.Stop(); err != nil {
		logger.Error("Event queue stop error", zap.Error(err))
	}
	m.queueCancel()

	// Close storage connection (database if enabled)
	if err := m.dualStore.Close(); err != nil {
		logger.Error("Storage close error", zap.Error(err))
	}

	// Close stop channel
	close(m.stopChan)

	logger.Info("Governance manager stopped")
	logger.Sync() // Flush any buffered logs
	return nil
}

// Wait blocks until the manager is stopped
func (m *Manager) Wait() {
	<-m.stopChan
}

// GetRegistry returns the registry (for testing/debugging)
func (m *Manager) GetRegistry() *registry.Registry {
	return m.registry
}

// GetConfig returns the manager configuration
func (m *Manager) GetConfig() *models.ManagerConfig {
	return m.config
}

// GetServicePods returns all pods for a given service group
func (m *Manager) GetServicePods(serviceName string) []*models.ServiceInfo {
	return m.registry.GetByServiceName(serviceName)
}

// GetAllServicePods returns a map of service names to their pods
func (m *Manager) GetAllServicePods() map[string][]*models.ServiceInfo {
	allServices := m.registry.GetAllServices()
	result := make(map[string][]*models.ServiceInfo)

	for _, service := range allServices {
		result[service.ServiceName] = append(result[service.ServiceName], service)
	}

	return result
}
