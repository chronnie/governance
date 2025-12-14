package models

import (
	"testing"
	"time"
)

func TestProtocolConstants(t *testing.T) {
	protocols := []Protocol{
		ProtocolHTTP,
		ProtocolTCP,
		ProtocolPFCP,
		ProtocolGTP,
		ProtocolUDP,
	}

	expectedValues := []string{"http", "tcp", "pfcp", "gtp", "udp"}

	for i, protocol := range protocols {
		if string(protocol) != expectedValues[i] {
			t.Errorf("Expected protocol '%s', got '%s'", expectedValues[i], protocol)
		}
	}
}

func TestServiceStatusConstants(t *testing.T) {
	statuses := []ServiceStatus{
		StatusHealthy,
		StatusUnhealthy,
		StatusUnknown,
	}

	expectedValues := []string{"healthy", "unhealthy", "unknown"}

	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Expected status '%s', got '%s'", expectedValues[i], status)
		}
	}
}

func TestEventTypeConstants(t *testing.T) {
	eventTypes := []EventType{
		EventTypeRegister,
		EventTypeUnregister,
		EventTypeUpdate,
		EventTypeReconcile,
	}

	expectedValues := []string{"register", "unregister", "update", "reconcile"}

	for i, eventType := range eventTypes {
		if string(eventType) != expectedValues[i] {
			t.Errorf("Expected event type '%s', got '%s'", expectedValues[i], eventType)
		}
	}
}

func TestServiceInfoGetKey(t *testing.T) {
	testCases := []struct {
		serviceName string
		podName     string
		expectedKey string
	}{
		{"service-a", "pod-1", "service-a:pod-1"},
		{"test", "instance-123", "test:instance-123"},
		{"my-service", "my-pod", "my-service:my-pod"},
	}

	for _, tc := range testCases {
		serviceInfo := &ServiceInfo{
			ServiceName: tc.serviceName,
			PodName:     tc.podName,
		}

		key := serviceInfo.GetKey()
		if key != tc.expectedKey {
			t.Errorf("Expected key '%s', got '%s'", tc.expectedKey, key)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Verify default values
	if config.ServerPort != 8080 {
		t.Errorf("Expected ServerPort 8080, got %d", config.ServerPort)
	}

	if config.HealthCheckInterval != 30*time.Second {
		t.Errorf("Expected HealthCheckInterval 30s, got %v", config.HealthCheckInterval)
	}

	if config.HealthCheckTimeout != 5*time.Second {
		t.Errorf("Expected HealthCheckTimeout 5s, got %v", config.HealthCheckTimeout)
	}

	if config.HealthCheckRetry != 3 {
		t.Errorf("Expected HealthCheckRetry 3, got %d", config.HealthCheckRetry)
	}

	if config.NotificationInterval != 60*time.Second {
		t.Errorf("Expected NotificationInterval 60s, got %v", config.NotificationInterval)
	}

	if config.NotificationTimeout != 5*time.Second {
		t.Errorf("Expected NotificationTimeout 5s, got %v", config.NotificationTimeout)
	}

	if config.EventQueueSize != 1000 {
		t.Errorf("Expected EventQueueSize 1000, got %d", config.EventQueueSize)
	}
}

func TestProviderInfoStruct(t *testing.T) {
	provider := ProviderInfo{
		Protocol: ProtocolHTTP,
		IP:       "192.168.1.10",
		Port:     8080,
	}

	if provider.Protocol != ProtocolHTTP {
		t.Errorf("Expected protocol 'http', got '%s'", provider.Protocol)
	}
	if provider.IP != "192.168.1.10" {
		t.Errorf("Expected IP '192.168.1.10', got '%s'", provider.IP)
	}
	if provider.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", provider.Port)
	}
}

func TestServiceRegistrationStruct(t *testing.T) {
	reg := ServiceRegistration{
		ServiceName: "test-service",
		PodName:     "test-pod",
		Providers: []ProviderInfo{
			{Protocol: ProtocolHTTP, IP: "192.168.1.10", Port: 8080},
		},
		HealthCheckURL:  "http://192.168.1.10:8080/health",
		NotificationURL: "http://192.168.1.10:8080/notify",
		Subscriptions:   []string{"service-a", "service-b"},
	}

	if reg.ServiceName != "test-service" {
		t.Error("ServiceName mismatch")
	}
	if len(reg.Providers) != 1 {
		t.Error("Providers length mismatch")
	}
	if len(reg.Subscriptions) != 2 {
		t.Error("Subscriptions length mismatch")
	}
}

func TestNotificationPayloadStruct(t *testing.T) {
	now := time.Now()
	payload := NotificationPayload{
		ServiceName: "test-service",
		EventType:   EventTypeRegister,
		Timestamp:   now,
		Pods: []PodInfo{
			{
				PodName: "pod-1",
				Status:  StatusHealthy,
				Providers: []ProviderInfo{
					{Protocol: ProtocolHTTP, IP: "192.168.1.10", Port: 8080},
				},
			},
		},
	}

	if payload.ServiceName != "test-service" {
		t.Error("ServiceName mismatch")
	}
	if payload.EventType != EventTypeRegister {
		t.Error("EventType mismatch")
	}
	if payload.Timestamp != now {
		t.Error("Timestamp mismatch")
	}
	if len(payload.Pods) != 1 {
		t.Error("Pods length mismatch")
	}
	if payload.Pods[0].PodName != "pod-1" {
		t.Error("PodName mismatch")
	}
}

func TestPodInfoStruct(t *testing.T) {
	podInfo := PodInfo{
		PodName: "test-pod",
		Status:  StatusHealthy,
		Providers: []ProviderInfo{
			{Protocol: ProtocolTCP, IP: "10.0.0.1", Port: 3000},
		},
	}

	if podInfo.PodName != "test-pod" {
		t.Error("PodName mismatch")
	}
	if podInfo.Status != StatusHealthy {
		t.Error("Status mismatch")
	}
	if len(podInfo.Providers) != 1 {
		t.Error("Providers length mismatch")
	}
}
