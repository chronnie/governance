package scheduler

import (
	"time"

	eventqueue "github.com/chronnie/go-event-queue"
	"github.com/chronnie/governance/events"
	"github.com/chronnie/governance/internal/registry"
	"github.com/chronnie/governance/pkg/logger"
	"go.uber.org/zap"
)

// HealthCheckScheduler periodically schedules health check events for all services
type HealthCheckScheduler struct {
	registry   *registry.Registry
	eventQueue eventqueue.IEventQueue
	interval   time.Duration
	stopChan   chan struct{}
}

// NewHealthCheckScheduler creates a new health check scheduler
func NewHealthCheckScheduler(reg *registry.Registry, eventQueue eventqueue.IEventQueue, interval time.Duration) *HealthCheckScheduler {
	return &HealthCheckScheduler{
		registry:   reg,
		eventQueue: eventQueue,
		interval:   interval,
		stopChan:   make(chan struct{}),
	}
}

// Start begins the health check scheduling
func (s *HealthCheckScheduler) Start() {
	logger.Info("HealthCheckScheduler: Starting health check scheduler",
		zap.Duration("interval", s.interval),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("HealthCheckScheduler: Ticker fired, scheduling health checks")
			s.scheduleHealthChecks()
		case <-s.stopChan:
			logger.Info("HealthCheckScheduler: Stopping health check scheduler")
			return
		}
	}
}

// Stop stops the health check scheduler
func (s *HealthCheckScheduler) Stop() {
	logger.Debug("HealthCheckScheduler: Stop signal sent")
	close(s.stopChan)
}

// scheduleHealthChecks creates health check events for all registered services
func (s *HealthCheckScheduler) scheduleHealthChecks() {
	services := s.registry.GetAllServices()

	logger.Debug("HealthCheckScheduler: Scheduling health checks for all services",
		zap.Int("service_count", len(services)),
	)

	for _, service := range services {
		logger.Debug("HealthCheckScheduler: Enqueuing health check event",
			zap.String("service_key", service.GetKey()),
			zap.String("service_name", service.ServiceName),
			zap.String("pod_name", service.PodName),
		)

		// Create context with event data
		ctx := events.NewHealthCheckContext(service.GetKey())

		// Create event (without deadline for health checks)
		event := eventqueue.NewEvent(string(events.EventHealthCheck), ctx)

		// Enqueue event
		s.eventQueue.Enqueue(event)
	}

	logger.Info("HealthCheckScheduler: Scheduled health checks",
		zap.Int("events_enqueued", len(services)),
	)
}

// ReconcileScheduler periodically schedules reconcile events
type ReconcileScheduler struct {
	eventQueue eventqueue.IEventQueue
	interval   time.Duration
	stopChan   chan struct{}
}

// NewReconcileScheduler creates a new reconcile scheduler
func NewReconcileScheduler(eventQueue eventqueue.IEventQueue, interval time.Duration) *ReconcileScheduler {
	return &ReconcileScheduler{
		eventQueue: eventQueue,
		interval:   interval,
		stopChan:   make(chan struct{}),
	}
}

// Start begins the reconcile scheduling
func (s *ReconcileScheduler) Start() {
	logger.Info("ReconcileScheduler: Starting reconcile scheduler",
		zap.Duration("interval", s.interval),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Debug("ReconcileScheduler: Ticker fired, scheduling reconcile")
			s.scheduleReconcile()
		case <-s.stopChan:
			logger.Info("ReconcileScheduler: Stopping reconcile scheduler")
			return
		}
	}
}

// Stop stops the reconcile scheduler
func (s *ReconcileScheduler) Stop() {
	logger.Debug("ReconcileScheduler: Stop signal sent")
	close(s.stopChan)
}

// scheduleReconcile creates a reconcile event
func (s *ReconcileScheduler) scheduleReconcile() {
	logger.Info("ReconcileScheduler: Enqueuing reconcile event")

	// Create context with event data
	ctx := events.NewReconcileContext()

	// Create event (without deadline for reconcile)
	event := eventqueue.NewEvent(string(events.EventReconcile), ctx)

	// Enqueue event
	s.eventQueue.Enqueue(event)

	logger.Debug("ReconcileScheduler: Reconcile event enqueued")
}
