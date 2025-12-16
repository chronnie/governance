package worker

import (
	"context"

	eventqueue "github.com/chronnie/go-event-queue"
	"github.com/chronnie/governance/events"
	"github.com/chronnie/governance/internal/notifier"
	"github.com/chronnie/governance/internal/registry"
	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/storage"
)

// EventWorker processes events from the queue using handlers
type EventWorker struct {
	registry      *registry.Registry
	notifier      *notifier.Notifier
	healthChecker *notifier.HealthChecker
	dualStore     *storage.DualStore // For database sync during reconciliation
}

// NewEventWorker creates a new event worker
func NewEventWorker(
	reg *registry.Registry,
	notif *notifier.Notifier,
	healthCheck *notifier.HealthChecker,
	dualStore *storage.DualStore,
) *EventWorker {
	return &EventWorker{
		registry:      reg,
		notifier:      notif,
		healthChecker: healthCheck,
		dualStore:     dualStore,
	}
}

// RegisterHandlers registers all event handlers to the queue
func (w *EventWorker) RegisterHandlers(queue eventqueue.IEventQueue) {
	// Register handler for each event type
	queue.RegisterHandler(string(events.EventRegister), eventqueue.EventHandlerFunc(w.handleRegister))
	queue.RegisterHandler(string(events.EventUnregister), eventqueue.EventHandlerFunc(w.handleUnregister))
	queue.RegisterHandler(string(events.EventHealthCheck), eventqueue.EventHandlerFunc(w.handleHealthCheck))
	queue.RegisterHandler(string(events.EventReconcile), eventqueue.EventHandlerFunc(w.handleReconcile))
}

// handleRegister processes service registration
func (w *EventWorker) handleRegister(ctx context.Context, event eventqueue.IEvent) error {
	eventData := events.GetEventData(ctx)
	registerEvent, ok := eventData.(*events.RegisterEvent)
	if !ok {
		return nil
	}

	// Register service in registry
	serviceInfo := w.registry.Register(registerEvent.Registration)

	// Get all pods of this service
	servicePods := w.registry.GetByServiceName(serviceInfo.ServiceName)

	// Build notification payload
	payload := notifier.BuildNotificationPayload(
		serviceInfo.ServiceName,
		models.EventTypeRegister,
		servicePods,
	)

	// Notify all subscribers of this service
	subscribers := w.registry.GetSubscriberServices(serviceInfo.ServiceName)
	w.notifier.NotifySubscribers(subscribers, payload)

	return nil
}

// handleUnregister processes service unregistration
func (w *EventWorker) handleUnregister(ctx context.Context, event eventqueue.IEvent) error {
	eventData := events.GetEventData(ctx)
	unregisterEvent, ok := eventData.(*events.UnregisterEvent)
	if !ok {
		return nil
	}

	// Unregister service from registry
	serviceInfo := w.registry.Unregister(unregisterEvent.ServiceName, unregisterEvent.PodName)
	if serviceInfo == nil {
		return nil
	}

	// Get remaining pods of this service (after unregistration)
	servicePods := w.registry.GetByServiceName(unregisterEvent.ServiceName)

	// Build notification payload
	payload := notifier.BuildNotificationPayload(
		unregisterEvent.ServiceName,
		models.EventTypeUnregister,
		servicePods,
	)

	// Notify all subscribers of this service
	subscribers := w.registry.GetSubscriberServices(unregisterEvent.ServiceName)
	w.notifier.NotifySubscribers(subscribers, payload)

	return nil
}

// handleHealthCheck processes health check event
func (w *EventWorker) handleHealthCheck(ctx context.Context, event eventqueue.IEvent) error {
	eventData := events.GetEventData(ctx)
	healthCheckEvent, ok := eventData.(*events.HealthCheckEvent)
	if !ok {
		return nil
	}

	// Get service from registry
	serviceInfo, exists := w.registry.Get(healthCheckEvent.ServiceKey)
	if !exists {
		return nil
	}

	// Perform health check with retries
	newStatus := w.healthChecker.GetHealthStatus(serviceInfo.HealthCheckURL)

	// Update health status in registry
	statusChanged := w.registry.UpdateHealthStatus(healthCheckEvent.ServiceKey, newStatus)

	// If status changed, notify subscribers
	if statusChanged {
		// Get all pods of this service
		servicePods := w.registry.GetByServiceName(serviceInfo.ServiceName)

		// Build notification payload
		payload := notifier.BuildNotificationPayload(
			serviceInfo.ServiceName,
			models.EventTypeUpdate,
			servicePods,
		)

		// Notify all subscribers
		subscribers := w.registry.GetSubscriberServices(serviceInfo.ServiceName)
		w.notifier.NotifySubscribers(subscribers, payload)
	}

	return nil
}

// handleReconcile processes reconcile event (notify all subscribers with current state + sync database)
func (w *EventWorker) handleReconcile(ctx context.Context, event eventqueue.IEvent) error {
	// Sync from database to cache (if database is enabled)
	// This ensures cache has the latest data from database
	if w.dualStore.GetDatabase() != nil {
		w.dualStore.SyncFromDatabase(ctx)
	}

	// Get all services from cache
	allServices := w.registry.GetAllServices()

	// Group services by service name
	serviceGroups := make(map[string][]*models.ServiceInfo)
	for _, service := range allServices {
		serviceGroups[service.ServiceName] = append(serviceGroups[service.ServiceName], service)
	}

	// For each service group, notify all subscribers
	for serviceName, pods := range serviceGroups {
		// Build notification payload
		payload := notifier.BuildNotificationPayload(
			serviceName,
			models.EventTypeReconcile,
			pods,
		)

		// Get subscribers
		subscribers := w.registry.GetSubscriberServices(serviceName)
		if len(subscribers) > 0 {
			w.notifier.NotifySubscribers(subscribers, payload)
		}
	}

	return nil
}
