package api

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/domain/ports/output"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Manual Mocks ---

type sentMessage struct {
	topic   string
	key     string
	message []byte
}

type mockKafkaAdapter struct {
	mu           sync.Mutex
	sendMessages []sentMessage
	readErr      error
	subscribeErr error
	commitErr    error
}

func (m *mockKafkaAdapter) SendMessage(topic, key string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendMessages = append(m.sendMessages, sentMessage{topic: topic, key: key, message: message})
	return nil
}

func (m *mockKafkaAdapter) ReadMessage() (*output.KafkaMessage, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	return &output.KafkaMessage{Value: []byte(`"test"`), Topic: "t"}, nil
}

func (m *mockKafkaAdapter) SubscribeTopics(topics []string) error {
	return m.subscribeErr
}

func (m *mockKafkaAdapter) CommitMessage(msg *output.KafkaMessage) error {
	return m.commitErr
}

func (m *mockKafkaAdapter) Close() {}

func (m *mockKafkaAdapter) getSentMessages() []sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]sentMessage, len(m.sendMessages))
	copy(result, m.sendMessages)
	return result
}

type mockMessageHandler struct {
	mu        sync.Mutex
	callCount int
	errors    []error
}

func (m *mockMessageHandler) HandleMessage(ctx context.Context, msg []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	idx := m.callCount
	m.callCount++
	if idx < len(m.errors) {
		return m.errors[idx]
	}
	return nil
}

func (m *mockMessageHandler) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// --- handleWithRetry tests ---

func TestHandleWithRetry_PermanentError_DLQImmediate(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	handler := &mockMessageHandler{
		errors: []error{domainerrors.NewPermanentError("bad data", nil)},
	}

	result := ks.handleWithRetry(context.Background(), handler, []byte(`"msg"`), "test-topic")

	assert.True(t, result, "should return true (DLQ counts as handled)")
	assert.Equal(t, 1, handler.getCallCount(), "handler should be called exactly once")

	msgs := adapter.getSentMessages()
	require.Len(t, msgs, 1, "should have sent one message to DLQ")
	assert.Equal(t, dlqTopic, msgs[0].topic)
}

func TestHandleWithRetry_TransientThenSuccess(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	handler := &mockMessageHandler{
		errors: []error{domainerrors.NewTransientError("temporary", nil)},
		// second call returns nil (success)
	}

	result := ks.handleWithRetry(context.Background(), handler, []byte(`"msg"`), "test-topic")

	assert.True(t, result)
	assert.Equal(t, 2, handler.getCallCount(), "handler called twice: initial + 1 retry")
	assert.Empty(t, adapter.getSentMessages(), "no DLQ message on successful retry")
}

func TestHandleWithRetry_AllRetriesExhausted_DLQ(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	transient := domainerrors.NewTransientError("always fails", nil)
	handler := &mockMessageHandler{
		errors: []error{transient, transient, transient, transient},
	}

	result := ks.handleWithRetry(context.Background(), handler, []byte(`"msg"`), "test-topic")

	assert.True(t, result, "should return true (DLQ counts as handled)")
	assert.Equal(t, 1+maxRetries, handler.getCallCount(), "handler called 1 initial + maxRetries times")

	msgs := adapter.getSentMessages()
	require.Len(t, msgs, 1, "should have sent one message to DLQ")
	assert.Equal(t, dlqTopic, msgs[0].topic)
}

// --- SendToDLQ tests ---

func TestSendToDLQ_JSONFormat(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	original := []byte(`{"plant":"alpha"}`)
	ks.SendToDLQ(original, "test error reason")

	msgs := adapter.getSentMessages()
	require.Len(t, msgs, 1)
	assert.Equal(t, dlqTopic, msgs[0].topic)

	var dlqMsg map[string]interface{}
	err := json.Unmarshal(msgs[0].message, &dlqMsg)
	require.NoError(t, err, "DLQ payload should be valid JSON")

	assert.Contains(t, dlqMsg, "original_message", "should have original_message field")
	assert.Contains(t, dlqMsg, "error_reason", "should have error_reason field")
	assert.Contains(t, dlqMsg, "timestamp", "should have timestamp field")
	assert.Equal(t, "test error reason", dlqMsg["error_reason"])
}

// --- StopConsuming tests ---

func TestStopConsuming_CalledTwice_NoPanic(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	assert.NotPanics(t, func() {
		ks.StopConsuming()
		ks.StopConsuming()
	})
}

// --- IsConsumerHealthy tests ---

func TestIsConsumerHealthy_TrueByDefault(t *testing.T) {
	adapter := &mockKafkaAdapter{}
	ks := NewKafkaService(adapter)

	assert.True(t, ks.IsConsumerHealthy())
}

func TestIsConsumerHealthy_FalseAfterConsumeEventsStopped(t *testing.T) {
	// readErr ensures ReadMessage returns immediately so the loop spins fast
	adapter := &mockKafkaAdapter{readErr: assert.AnError}
	ks := NewKafkaService(adapter)

	// Register a topic so ConsumeEvents doesn't exit immediately
	ks.RegisterHandler("t", &mockMessageHandler{})

	done := make(chan struct{})
	go func() {
		ks.ConsumeEvents()
		close(done)
	}()

	ks.StopConsuming()
	<-done // wait for goroutine to finish cleanly

	assert.False(t, ks.IsConsumerHealthy())
}
