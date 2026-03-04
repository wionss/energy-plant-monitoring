package services

import (
	"testing"

	"monitoring-energy-service/internal/infrastructure/adapters/telegram"

	"github.com/stretchr/testify/assert"
)

func newTestService(t *testing.T) *AlertEvaluationService {
	n := telegram.NewNotifier("", "", false)
	t.Cleanup(func() { n.Stop() })
	return NewAlertEvaluationService(n)
}

// --- toFloat64 tests ---

func TestToFloat64_Float64(t *testing.T) {
	v, ok := toFloat64(float64(3.14))
	assert.True(t, ok)
	assert.InDelta(t, 3.14, v, 0.0001)
}

func TestToFloat64_Float32(t *testing.T) {
	v, ok := toFloat64(float32(1.5))
	assert.True(t, ok)
	assert.InDelta(t, 1.5, v, 0.0001)
}

func TestToFloat64_Int(t *testing.T) {
	v, ok := toFloat64(int(42))
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestToFloat64_Int32(t *testing.T) {
	v, ok := toFloat64(int32(7))
	assert.True(t, ok)
	assert.Equal(t, float64(7), v)
}

func TestToFloat64_Int64(t *testing.T) {
	v, ok := toFloat64(int64(100))
	assert.True(t, ok)
	assert.Equal(t, float64(100), v)
}

func TestToFloat64_StringNumeric(t *testing.T) {
	v, ok := toFloat64("3.14")
	assert.True(t, ok)
	assert.InDelta(t, 3.14, v, 0.0001)
}

func TestToFloat64_StringNonNumeric(t *testing.T) {
	_, ok := toFloat64("abc")
	assert.False(t, ok)
}

func TestToFloat64_UnknownType(t *testing.T) {
	_, ok := toFloat64(true)
	assert.False(t, ok)
}

// --- containsString tests ---

func TestContainsString_Match(t *testing.T) {
	assert.True(t, containsString("hello world", "world"))
}

func TestContainsString_NoMatch(t *testing.T) {
	assert.False(t, containsString("hello world", "xyz"))
}

func TestContainsString_NonString(t *testing.T) {
	assert.False(t, containsString(int(42), "42"))
}

// --- evaluateCondition tests ---

func TestEvaluateCondition_GtTrue(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "gt", FieldPath: "val", Threshold: 10}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(15)}))
}

func TestEvaluateCondition_GtFalse(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "gt", FieldPath: "val", Threshold: 10}
	assert.False(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(5)}))
}

func TestEvaluateCondition_Gte(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "gte", FieldPath: "val", Threshold: 10}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(10)}))
}

func TestEvaluateCondition_Lt(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "lt", FieldPath: "val", Threshold: 10}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(5)}))
}

func TestEvaluateCondition_Lte(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "lte", FieldPath: "val", Threshold: 10}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(10)}))
}

func TestEvaluateCondition_Eq(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "eq", FieldPath: "val", Threshold: 42}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(42)}))
}

func TestEvaluateCondition_ContainsWithThresholdStr(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "contains", FieldPath: "msg", ThresholdStr: "error"}
	assert.True(t, svc.evaluateCondition(rule, map[string]interface{}{"msg": "critical error occurred"}))
}

func TestEvaluateCondition_UnknownCondition(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "unknown", FieldPath: "val", Threshold: 1}
	assert.False(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(99)}))
}

func TestEvaluateCondition_FieldNotFound(t *testing.T) {
	svc := newTestService(t)
	rule := AlertRule{Condition: "gt", FieldPath: "missing", Threshold: 10}
	assert.False(t, svc.evaluateCondition(rule, map[string]interface{}{"val": float64(99)}))
}

// --- EvaluateEvent tests ---

func TestEvaluateEvent_NoMatchingRule(t *testing.T) {
	svc := newTestService(t)
	// Should not panic with no matching rule
	svc.EvaluateEvent("id1", "unknown_type", "plant1", "Plant One", map[string]interface{}{
		"temperature": float64(200),
	})
}

func TestEvaluateEvent_MatchingRuleNilNotifier(t *testing.T) {
	svc := &AlertEvaluationService{
		telegramNotifier: nil,
		rules:            getDefaultAlertRules(),
	}
	// Should not panic even with nil notifier
	svc.EvaluateEvent("id2", "temperature", "plant1", "Plant One", map[string]interface{}{
		"temperature": float64(90),
	})
}

func TestEvaluateEvent_MatchingRuleDisabledNotifier(t *testing.T) {
	svc := newTestService(t) // notifier is disabled (enabled=false)
	// Should complete without error
	svc.EvaluateEvent("id3", "temperature", "plant1", "Plant One", map[string]interface{}{
		"temperature": float64(90),
	})
}
