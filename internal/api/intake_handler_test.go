package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"monitoring-energy-service/internal/domain/entities"
	domainerrors "monitoring-energy-service/internal/domain/errors"
	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Manual Mocks ---

// mockDualEventWriter implements output.DualEventWriterInterface
type mockDualEventWriter struct {
	saveEventCalled      bool
	saveEventAsyncCalled bool
	savedOp              *entities.EventOperational
	savedAn              *entities.EventAnalytical
	saveErr              error
}

func (m *mockDualEventWriter) SaveEvent(op *entities.EventOperational, an *entities.EventAnalytical) error {
	m.saveEventCalled = true
	m.savedOp = op
	m.savedAn = an
	return m.saveErr
}

func (m *mockDualEventWriter) SaveEventAsync(op *entities.EventOperational, an *entities.EventAnalytical) error {
	m.saveEventAsyncCalled = true
	m.savedOp = op
	m.savedAn = an
	return m.saveErr
}

func (m *mockDualEventWriter) Stop() {}

// mockEnergyPlantRepo implements output.EnergyPlantRepositoryInterface
type mockEnergyPlantRepo struct {
	existsResult bool
	existsErr    error
	findResult   *entities.EnergyPlants
	findErr      error
}

func (m *mockEnergyPlantRepo) FindByID(id uuid.UUID) (*entities.EnergyPlants, error) {
	return m.findResult, m.findErr
}

func (m *mockEnergyPlantRepo) Exists(id uuid.UUID) (bool, error) {
	return m.existsResult, m.existsErr
}

// mockPlantStatusRepo implements output.PlantStatusRepositoryInterface
type mockPlantStatusRepo struct {
	upsertCalled bool
	upsertErr    error
}

func (m *mockPlantStatusRepo) Upsert(status *entities.PlantCurrentStatus) error {
	m.upsertCalled = true
	return m.upsertErr
}

func (m *mockPlantStatusRepo) GetByPlantID(plantID uuid.UUID) (*entities.PlantCurrentStatus, error) {
	return nil, nil
}

func (m *mockPlantStatusRepo) GetAll() ([]*entities.PlantCurrentStatus, error) {
	return nil, nil
}

func (m *mockPlantStatusRepo) GetByStatus(status string) ([]*entities.PlantCurrentStatus, error) {
	return nil, nil
}

// --- Test Helpers ---

var testPlantID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func validMessage() []byte {
	msg := map[string]any{
		"event_type":      "power_reading",
		"plant_name":      "Solar Farm Alpha",
		"plant_source_id": testPlantID.String(),
		"power_generated_mw":  50.5,
		"power_consumed_mw":   10.2,
		"efficiency_percent":  85.0,
		"temperature_celsius": 35.0,
	}
	data, _ := json.Marshal(msg)
	return data
}

func newHandler(
	writer *mockDualEventWriter,
	plantRepo *mockEnergyPlantRepo,
	statusRepo *mockPlantStatusRepo,
	asyncWrite bool,
) *IntakeHandler {
	notifier := telegram.NewNotifier("", "", false)
	return NewIntakeHandler(writer, plantRepo, statusRepo, notifier, asyncWrite)
}

// --- Tests ---

func TestHandleMessage_ValidMessage_AsyncWrite(t *testing.T) {
	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(validMessage())

	require.NoError(t, err)
	assert.True(t, writer.saveEventAsyncCalled, "SaveEventAsync should be called")
	assert.False(t, writer.saveEventCalled, "SaveEvent should NOT be called in async mode")
	assert.Equal(t, "power_reading", writer.savedOp.EventType)
	assert.Equal(t, testPlantID, writer.savedOp.PlantSourceId)
	assert.Equal(t, "Solar Farm Alpha", writer.savedOp.Source)
	assert.True(t, statusRepo.upsertCalled, "plant status should be updated")
}

func TestHandleMessage_ValidMessage_SyncWrite(t *testing.T) {
	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, false)

	err := handler.HandleMessage(validMessage())

	require.NoError(t, err)
	assert.True(t, writer.saveEventCalled, "SaveEvent should be called in sync mode")
	assert.False(t, writer.saveEventAsyncCalled, "SaveEventAsync should NOT be called in sync mode")
}

func TestHandleMessage_InvalidJSON(t *testing.T) {
	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage([]byte("not valid json{{{"))

	require.Error(t, err)
	assert.True(t, domainerrors.IsPermanent(err), "invalid JSON should be a permanent error")
	assert.False(t, writer.saveEventAsyncCalled, "should not save on invalid JSON")
}

func TestHandleMessage_MissingPlantSourceID(t *testing.T) {
	msg := map[string]any{
		"event_type": "power_reading",
		"plant_name": "Test Plant",
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.Error(t, err)
	assert.True(t, domainerrors.IsPermanent(err), "missing plant_source_id should be permanent error")
}

func TestHandleMessage_InvalidPlantSourceIDFormat(t *testing.T) {
	msg := map[string]any{
		"event_type":      "power_reading",
		"plant_source_id": "not-a-uuid",
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.Error(t, err)
	assert.True(t, domainerrors.IsPermanent(err), "invalid UUID format should be permanent error")
}

func TestHandleMessage_PlantDoesNotExist(t *testing.T) {
	msg := map[string]any{
		"event_type":      "power_reading",
		"plant_source_id": testPlantID.String(),
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: false}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.Error(t, err)
	assert.True(t, domainerrors.IsTransient(err), "non-existent plant should be transient error (may appear later)")
}

func TestHandleMessage_PlantExistsCheckDBError(t *testing.T) {
	msg := map[string]any{
		"event_type":      "power_reading",
		"plant_source_id": testPlantID.String(),
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsErr: fmt.Errorf("connection refused")}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.Error(t, err)
	assert.True(t, domainerrors.IsTransient(err), "DB error should be transient (retryable)")
}

func TestHandleMessage_SaveAsyncError(t *testing.T) {
	writer := &mockDualEventWriter{saveErr: fmt.Errorf("channel full")}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(validMessage())

	require.Error(t, err)
	assert.True(t, domainerrors.IsTransient(err), "save error should be transient")
}

func TestHandleMessage_ValidationFailure(t *testing.T) {
	msg := map[string]any{
		"event_type":        "power_reading",
		"plant_source_id":   testPlantID.String(),
		"efficiency_percent": 150.0, // Invalid: exceeds 100
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.Error(t, err)
	assert.True(t, domainerrors.IsPermanent(err), "validation failure should be permanent error")
	assert.False(t, writer.saveEventAsyncCalled, "should not save on validation error")
}

func TestHandleMessage_IdempotentID(t *testing.T) {
	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	message := validMessage()

	// Process same message twice
	err1 := handler.HandleMessage(message)
	require.NoError(t, err1)
	firstID := writer.savedOp.ID

	err2 := handler.HandleMessage(message)
	require.NoError(t, err2)
	secondID := writer.savedOp.ID

	assert.Equal(t, firstID, secondID, "same message should produce the same deterministic ID")
}

func TestHandleMessage_ExplicitIDInMessage(t *testing.T) {
	explicitID := uuid.New()
	msg := map[string]any{
		"id":              explicitID.String(),
		"event_type":      "power_reading",
		"plant_name":      "Test Plant",
		"plant_source_id": testPlantID.String(),
	}
	data, _ := json.Marshal(msg)

	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(data)

	require.NoError(t, err)
	assert.Equal(t, explicitID, writer.savedOp.ID, "should use the explicit ID from message")
}

func TestHandleMessage_PlantStatusUpsertFailure_NonBlocking(t *testing.T) {
	writer := &mockDualEventWriter{}
	plantRepo := &mockEnergyPlantRepo{existsResult: true}
	statusRepo := &mockPlantStatusRepo{upsertErr: fmt.Errorf("DB error")}
	handler := newHandler(writer, plantRepo, statusRepo, true)

	err := handler.HandleMessage(validMessage())

	// Message processing should succeed even if plant status update fails
	require.NoError(t, err)
	assert.True(t, writer.saveEventAsyncCalled, "event should still be saved")
	assert.True(t, statusRepo.upsertCalled, "upsert should have been attempted")
}
