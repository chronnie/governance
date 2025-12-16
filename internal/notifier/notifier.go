package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/chronnie/governance/models"
	"github.com/chronnie/governance/pkg/logger"
	"go.uber.org/zap"
)

// Notifier handles sending notifications to subscribers
type Notifier struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewNotifier creates a new notifier with given timeout
func NewNotifier(timeout time.Duration) *Notifier {
	return &Notifier{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// NotifySubscribers sends notification to all subscribers
// Does not retry on failure as per requirements
func (n *Notifier) NotifySubscribers(subscribers []*models.ServiceInfo, payload *models.NotificationPayload) {
	logger.Debug("Notifier: NotifySubscribers called",
		zap.Int("subscriber_count", len(subscribers)),
		zap.String("event_type", string(payload.EventType)),
		zap.String("service_name", payload.ServiceName),
	)

	for _, subscriber := range subscribers {
		logger.Debug("Notifier: Sending notification to subscriber",
			zap.String("subscriber_key", subscriber.GetKey()),
			zap.String("notification_url", subscriber.NotificationURL),
			zap.String("event_type", string(payload.EventType)),
		)
		go n.sendNotification(subscriber.NotificationURL, payload, subscriber.GetKey())
	}
}

// NotifySubscriber sends notification to a single subscriber
func (n *Notifier) NotifySubscriber(notificationURL string, payload *models.NotificationPayload) {
	logger.Debug("Notifier: NotifySubscriber called",
		zap.String("notification_url", notificationURL),
		zap.String("event_type", string(payload.EventType)),
	)
	go n.sendNotification(notificationURL, payload, "")
}

// sendNotification sends HTTP POST notification to a URL
func (n *Notifier) sendNotification(url string, payload *models.NotificationPayload, subscriberKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), n.timeout)
	defer cancel()

	logFields := []zap.Field{
		zap.String("notification_url", url),
		zap.String("event_type", string(payload.EventType)),
		zap.String("service_name", payload.ServiceName),
	}
	if subscriberKey != "" {
		logFields = append(logFields, zap.String("subscriber_key", subscriberKey))
	}

	logger.Debug("Notifier: Sending HTTP POST notification", logFields...)

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Notifier: Failed to marshal notification payload",
			append(logFields, zap.Error(err))...)
		return
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Notifier: Failed to create notification request",
			append(logFields, zap.Error(err))...)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		logger.Error("Notifier: Failed to send notification",
			append(logFields, zap.Error(err))...)
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("Notifier: Notification returned non-success status",
			append(logFields, zap.Int("status_code", resp.StatusCode))...)
		return
	}

	logger.Info("Notifier: Successfully sent notification",
		append(logFields, zap.Int("status_code", resp.StatusCode))...)
}

// BuildNotificationPayload creates a notification payload from service pods
func BuildNotificationPayload(serviceName string, eventType models.EventType, pods []*models.ServiceInfo) *models.NotificationPayload {
	podInfos := make([]models.PodInfo, 0, len(pods))

	for _, pod := range pods {
		podInfos = append(podInfos, models.PodInfo{
			PodName:   pod.PodName,
			Status:    pod.Status,
			Providers: pod.Providers,
		})
	}

	return &models.NotificationPayload{
		ServiceName: serviceName,
		EventType:   eventType,
		Timestamp:   time.Now(),
		Pods:        podInfos,
	}
}

// HealthChecker performs health checks on services
type HealthChecker struct {
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration, maxRetries int) *HealthChecker {
	return &HealthChecker{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:    timeout,
		maxRetries: maxRetries,
	}
}

// CheckHealth performs health check with retries
// Returns true if healthy, false if unhealthy
func (hc *HealthChecker) CheckHealth(healthCheckURL string) bool {
	logger.Debug("HealthChecker: Starting health check",
		zap.String("health_check_url", healthCheckURL),
		zap.Int("max_retries", hc.maxRetries),
		zap.Duration("timeout", hc.timeout),
	)

	for attempt := 0; attempt <= hc.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s...
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			logger.Debug("HealthChecker: Retrying after backoff",
				zap.String("health_check_url", healthCheckURL),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", hc.maxRetries),
				zap.Duration("backoff", backoff),
			)
			time.Sleep(backoff)
		}

		ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthCheckURL, nil)
		if err != nil {
			cancel()
			logger.Error("HealthChecker: Failed to create health check request",
				zap.String("health_check_url", healthCheckURL),
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			continue
		}

		resp, err := hc.httpClient.Do(req)
		cancel()

		if err != nil {
			logger.Warn("HealthChecker: Health check request failed",
				zap.String("health_check_url", healthCheckURL),
				zap.Int("attempt", attempt+1),
				zap.Int("total_attempts", hc.maxRetries+1),
				zap.Error(err),
			)
			continue
		}

		resp.Body.Close()

		// Consider 2xx as healthy
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			logger.Debug("HealthChecker: Health check passed",
				zap.String("health_check_url", healthCheckURL),
				zap.Int("status_code", resp.StatusCode),
				zap.Int("attempt", attempt+1),
			)
			return true
		}

		logger.Warn("HealthChecker: Health check returned unhealthy status",
			zap.String("health_check_url", healthCheckURL),
			zap.Int("attempt", attempt+1),
			zap.Int("total_attempts", hc.maxRetries+1),
			zap.Int("status_code", resp.StatusCode),
		)
	}

	logger.Error("HealthChecker: Health check failed after all retries",
		zap.String("health_check_url", healthCheckURL),
		zap.Int("total_attempts", hc.maxRetries+1),
	)
	return false
}

// GetHealthStatus performs health check and returns status
func (hc *HealthChecker) GetHealthStatus(healthCheckURL string) models.ServiceStatus {
	if hc.CheckHealth(healthCheckURL) {
		return models.StatusHealthy
	}
	return models.StatusUnhealthy
}
