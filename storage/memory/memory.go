package memory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/storage"
)

// MemoryStore implements storage.RegistryStore using in-memory maps.
// This is the default storage implementation and is lock-free since
// it's accessed only by the single event queue worker.
type MemoryStore struct {
	services      map[string]*models.ServiceInfo // Key: "serviceName:podName"
	subscriptions map[string][]string            // Key: serviceGroup, Value: list of subscriber keys
}

// Ensure MemoryStore implements RegistryStore
var _ storage.RegistryStore = (*MemoryStore)(nil)

// NewMemoryStore creates a new in-memory storage instance
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		services:      make(map[string]*models.ServiceInfo),
		subscriptions: make(map[string][]string),
	}
}

// SaveService stores or updates a service entry
func (m *MemoryStore) SaveService(ctx context.Context, service *models.ServiceInfo) error {
	if service == nil {
		return errors.New("service cannot be nil")
	}

	key := service.GetKey()
	if key == "" {
		return errors.New("service key cannot be empty")
	}

	// Store a copy to avoid external mutations
	serviceCopy := *service
	m.services[key] = &serviceCopy

	return nil
}

// GetService retrieves a single service by its composite key
func (m *MemoryStore) GetService(ctx context.Context, key string) (*models.ServiceInfo, error) {
	service, exists := m.services[key]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", key)
	}

	// Return a copy to avoid external mutations
	serviceCopy := *service
	return &serviceCopy, nil
}

// GetServicesByName retrieves all pods for a given service name
func (m *MemoryStore) GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error) {
	var result []*models.ServiceInfo

	for _, service := range m.services {
		if service.ServiceName == serviceName {
			serviceCopy := *service
			result = append(result, &serviceCopy)
		}
	}

	return result, nil
}

// GetAllServices retrieves all registered services
func (m *MemoryStore) GetAllServices(ctx context.Context) ([]*models.ServiceInfo, error) {
	result := make([]*models.ServiceInfo, 0, len(m.services))

	for _, service := range m.services {
		serviceCopy := *service
		result = append(result, &serviceCopy)
	}

	return result, nil
}

// DeleteService removes a service entry by its composite key
func (m *MemoryStore) DeleteService(ctx context.Context, key string) error {
	if _, exists := m.services[key]; !exists {
		return fmt.Errorf("service not found: %s", key)
	}

	delete(m.services, key)
	return nil
}

// UpdateHealthStatus updates the health status and last check timestamp
func (m *MemoryStore) UpdateHealthStatus(ctx context.Context, key string, status models.ServiceStatus, timestamp time.Time) error {
	service, exists := m.services[key]
	if !exists {
		return fmt.Errorf("service not found: %s", key)
	}

	service.Status = status
	service.LastHealthCheck = timestamp

	return nil
}

// AddSubscription adds a subscriber to a service group
func (m *MemoryStore) AddSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error {
	if subscriberKey == "" {
		return errors.New("subscriber key cannot be empty")
	}
	if serviceGroup == "" {
		return errors.New("service group cannot be empty")
	}

	subscribers := m.subscriptions[serviceGroup]

	// Check if already subscribed
	for _, sub := range subscribers {
		if sub == subscriberKey {
			return nil // Already subscribed
		}
	}

	// Add new subscription
	m.subscriptions[serviceGroup] = append(subscribers, subscriberKey)
	return nil
}

// RemoveSubscription removes a subscriber from a service group
func (m *MemoryStore) RemoveSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error {
	subscribers, exists := m.subscriptions[serviceGroup]
	if !exists {
		return nil // No subscriptions for this service group
	}

	// Find and remove the subscriber
	for i, sub := range subscribers {
		if sub == subscriberKey {
			m.subscriptions[serviceGroup] = append(subscribers[:i], subscribers[i+1:]...)

			// Clean up empty subscription lists
			if len(m.subscriptions[serviceGroup]) == 0 {
				delete(m.subscriptions, serviceGroup)
			}

			return nil
		}
	}

	return nil // Subscriber not found, but that's okay
}

// RemoveAllSubscriptions removes all subscriptions for a given subscriber
func (m *MemoryStore) RemoveAllSubscriptions(ctx context.Context, subscriberKey string) error {
	for serviceGroup := range m.subscriptions {
		m.RemoveSubscription(ctx, subscriberKey, serviceGroup)
	}
	return nil
}

// GetSubscribers returns all subscriber keys for a given service group
func (m *MemoryStore) GetSubscribers(ctx context.Context, serviceGroup string) ([]string, error) {
	subscribers, exists := m.subscriptions[serviceGroup]
	if !exists {
		return []string{}, nil
	}

	// Return a copy to avoid external mutations
	result := make([]string, len(subscribers))
	copy(result, subscribers)

	return result, nil
}

// GetSubscriberServices returns full ServiceInfo objects for all subscribers
func (m *MemoryStore) GetSubscriberServices(ctx context.Context, serviceGroup string) ([]*models.ServiceInfo, error) {
	subscribers, err := m.GetSubscribers(ctx, serviceGroup)
	if err != nil {
		return nil, err
	}

	result := make([]*models.ServiceInfo, 0, len(subscribers))

	for _, subscriberKey := range subscribers {
		service, err := m.GetService(ctx, subscriberKey)
		if err != nil {
			// Skip services that no longer exist
			continue
		}
		result = append(result, service)
	}

	return result, nil
}

// Close is a no-op for memory storage
func (m *MemoryStore) Close() error {
	return nil
}

// Ping always succeeds for memory storage
func (m *MemoryStore) Ping(ctx context.Context) error {
	return nil
}
