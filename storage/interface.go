package storage

import (
	"context"
	"time"

	"github.com/chronnie/governance/models"
)

// RegistryStore defines the interface for persisting service registry data.
// Implementations can use different storage backends (memory, MySQL, PostgreSQL, MongoDB, etc.)
type RegistryStore interface {
	// Service operations

	// SaveService stores or updates a service entry
	SaveService(ctx context.Context, service *models.ServiceInfo) error

	// GetService retrieves a single service by its composite key (serviceName:podName)
	GetService(ctx context.Context, key string) (*models.ServiceInfo, error)

	// GetServicesByName retrieves all pods for a given service name
	GetServicesByName(ctx context.Context, serviceName string) ([]*models.ServiceInfo, error)

	// GetAllServices retrieves all registered services across all service groups
	GetAllServices(ctx context.Context) ([]*models.ServiceInfo, error)

	// DeleteService removes a service entry by its composite key
	DeleteService(ctx context.Context, key string) error

	// UpdateHealthStatus updates the health status and last check timestamp for a service
	UpdateHealthStatus(ctx context.Context, key string, status models.ServiceStatus, timestamp time.Time) error

	// Subscription operations

	// AddSubscription adds a subscriber to a service group
	// subscriberKey is the composite key (serviceName:podName) of the subscriber
	// serviceGroup is the name of the service being subscribed to
	AddSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error

	// RemoveSubscription removes a subscriber from a service group
	RemoveSubscription(ctx context.Context, subscriberKey string, serviceGroup string) error

	// RemoveAllSubscriptions removes all subscriptions for a given subscriber
	RemoveAllSubscriptions(ctx context.Context, subscriberKey string) error

	// GetSubscribers returns all subscriber keys for a given service group
	GetSubscribers(ctx context.Context, serviceGroup string) ([]string, error)

	// GetSubscriberServices returns full ServiceInfo objects for all subscribers of a service group
	GetSubscriberServices(ctx context.Context, serviceGroup string) ([]*models.ServiceInfo, error)

	// Lifecycle operations

	// Close closes the storage connection and cleans up resources
	Close() error

	// Ping checks if the storage backend is accessible
	Ping(ctx context.Context) error
}
