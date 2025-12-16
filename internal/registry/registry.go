package registry

import (
	"context"
	"time"

	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/storage"
)

// Registry manages all registered services using a pluggable storage backend
// No locks needed because it's accessed only by single event queue worker
type Registry struct {
	store storage.RegistryStore
	ctx   context.Context
}

// NewRegistry creates a new registry with the given storage backend
func NewRegistry(store storage.RegistryStore) *Registry {
	return &Registry{
		store: store,
		ctx:   context.Background(),
	}
}

// Register adds or updates a service in the registry
func (r *Registry) Register(reg *models.ServiceRegistration) *models.ServiceInfo {
	serviceInfo := &models.ServiceInfo{
		ServiceName:     reg.ServiceName,
		PodName:         reg.PodName,
		Providers:       reg.Providers,
		HealthCheckURL:  reg.HealthCheckURL,
		NotificationURL: reg.NotificationURL,
		Subscriptions:   reg.Subscriptions,
		Status:          models.StatusUnknown, // Initial status is unknown
		RegisteredAt:    time.Now(),
		LastHealthCheck: time.Time{},
	}

	key := serviceInfo.GetKey()

	// Remove old subscriptions if service already exists
	if oldService, err := r.store.GetService(r.ctx, key); err == nil {
		r.removeSubscriptions(key, oldService.Subscriptions)
	}

	// Save service to storage
	if err := r.store.SaveService(r.ctx, serviceInfo); err != nil {
		// Log error but continue (non-critical failure)
		return serviceInfo
	}

	// Add new subscriptions
	r.addSubscriptions(key, reg.Subscriptions)

	return serviceInfo
}

// Unregister removes a service from the registry
func (r *Registry) Unregister(serviceName, podName string) *models.ServiceInfo {
	key := serviceName + ":" + podName

	service, err := r.store.GetService(r.ctx, key)
	if err != nil {
		return nil
	}

	// Remove subscriptions
	r.removeSubscriptions(key, service.Subscriptions)

	// Remove from storage
	r.store.DeleteService(r.ctx, key)

	return service
}

// Get retrieves a service by key
func (r *Registry) Get(key string) (*models.ServiceInfo, bool) {
	service, err := r.store.GetService(r.ctx, key)
	if err != nil {
		return nil, false
	}
	return service, true
}

// GetByServiceName returns all pods of a service
func (r *Registry) GetByServiceName(serviceName string) []*models.ServiceInfo {
	result, err := r.store.GetServicesByName(r.ctx, serviceName)
	if err != nil {
		return []*models.ServiceInfo{}
	}
	return result
}

// GetAllServices returns all registered services
func (r *Registry) GetAllServices() []*models.ServiceInfo {
	result, err := r.store.GetAllServices(r.ctx)
	if err != nil {
		return []*models.ServiceInfo{}
	}
	return result
}

// UpdateHealthStatus updates the health status of a service
func (r *Registry) UpdateHealthStatus(key string, status models.ServiceStatus) bool {
	service, err := r.store.GetService(r.ctx, key)
	if err != nil {
		return false
	}

	oldStatus := service.Status
	timestamp := time.Now()

	// Update in storage
	if err := r.store.UpdateHealthStatus(r.ctx, key, status, timestamp); err != nil {
		return false
	}

	// Return true if status changed
	return oldStatus != status
}

// GetSubscribers returns all subscriber keys for a given service name
func (r *Registry) GetSubscribers(serviceName string) []string {
	subscribers, err := r.store.GetSubscribers(r.ctx, serviceName)
	if err != nil {
		return []string{}
	}
	return subscribers
}

// GetSubscriberServices returns all ServiceInfo of subscribers for a given service name
func (r *Registry) GetSubscriberServices(serviceName string) []*models.ServiceInfo {
	result, err := r.store.GetSubscriberServices(r.ctx, serviceName)
	if err != nil {
		return []*models.ServiceInfo{}
	}
	return result
}

// addSubscriptions adds subscriptions for a service
func (r *Registry) addSubscriptions(subscriberKey string, subscriptions []string) {
	for _, serviceName := range subscriptions {
		r.store.AddSubscription(r.ctx, subscriberKey, serviceName)
	}
}

// removeSubscriptions removes subscriptions for a service
func (r *Registry) removeSubscriptions(subscriberKey string, subscriptions []string) {
	for _, serviceName := range subscriptions {
		r.store.RemoveSubscription(r.ctx, subscriberKey, serviceName)
	}
}
