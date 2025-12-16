package storage

import (
	"context"
	"time"

	"github.com/chronnie/governance/models"
)

// DatabaseStore defines the interface for database persistence layer.
// This is simpler than RegistryStore - it's only for persistence, not for runtime queries.
// The in-memory cache handles all runtime operations.
type DatabaseStore interface {
	// Service persistence operations

	// SaveService stores or updates a service entry in the database
	SaveService(ctx context.Context, service *models.ServiceInfo) error

	// GetService retrieves a single service by its composite key (serviceName:podName)
	GetService(ctx context.Context, key string) (*models.ServiceInfo, error)

	// GetAllServices retrieves all registered services from database
	// Used during reconciliation to sync with cache
	GetAllServices(ctx context.Context) ([]*models.ServiceInfo, error)

	// DeleteService removes a service entry by its composite key
	DeleteService(ctx context.Context, key string) error

	// UpdateHealthStatus updates the health status and last check timestamp
	UpdateHealthStatus(ctx context.Context, key string, status models.ServiceStatus, timestamp time.Time) error

	// Subscription persistence operations

	// SaveSubscriptions saves all subscriptions for a service
	// This replaces all existing subscriptions for the given subscriber
	SaveSubscriptions(ctx context.Context, subscriberKey string, subscriptions []string) error

	// GetSubscriptions retrieves all service groups that a subscriber is subscribed to
	GetSubscriptions(ctx context.Context, subscriberKey string) ([]string, error)

	// GetAllSubscriptions retrieves all subscription relationships from database
	// Used during reconciliation to sync with cache
	// Returns map[subscriberKey][]serviceGroups
	GetAllSubscriptions(ctx context.Context) (map[string][]string, error)

	// DeleteSubscriptions removes all subscriptions for a subscriber
	DeleteSubscriptions(ctx context.Context, subscriberKey string) error

	// Lifecycle operations

	// Close closes the database connection and cleans up resources
	Close() error

	// Ping checks if the database is accessible
	Ping(ctx context.Context) error
}
