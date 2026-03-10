package repositories

import (
	"monitoring-energy-service/internal/domain/entities"
	"monitoring-energy-service/internal/domain/ports/output"

	"gorm.io/gorm"
)

// AlertRulesRepository fetches alert rules from the database.
type AlertRulesRepository struct {
	db *gorm.DB
}

var _ output.AlertRulesRepositoryInterface = &AlertRulesRepository{}

// NewAlertRulesRepository creates a new AlertRulesRepository.
func NewAlertRulesRepository(db *gorm.DB) *AlertRulesRepository {
	return &AlertRulesRepository{db: db}
}

// FindActive returns all active alert rules.
func (r *AlertRulesRepository) FindActive() ([]entities.AlertRule, error) {
	var models []AlertRuleModel
	if err := r.db.Where("active = ?", true).Find(&models).Error; err != nil {
		return nil, err
	}
	rules := make([]entities.AlertRule, len(models))
	for i, m := range models {
		rules[i] = ToAlertRuleFromModel(&m)
	}
	return rules, nil
}
