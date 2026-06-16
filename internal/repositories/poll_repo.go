package repositories

import (
	"time"

	"github.com/embrionix/dashboard/internal/models"
	"gorm.io/gorm"
)

type PollRepository struct {
	db *gorm.DB
}

func NewPollRepository(db *gorm.DB) *PollRepository {
	return &PollRepository{db: db}
}

func (r *PollRepository) Save(result *models.PollResult) error {
	return r.db.Create(result).Error
}

func (r *PollRepository) FindByDevice(deviceID string, limit int) ([]models.PollResult, error) {
	var results []models.PollResult
	q := r.db.Where("device_id = ?", deviceID).Order("polled_at desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	return results, q.Find(&results).Error
}

func (r *PollRepository) FindLatestByDevice(deviceID string) (*models.PollResult, error) {
	var result models.PollResult
	err := r.db.Where("device_id = ?", deviceID).Order("polled_at desc").First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *PollRepository) FindByDeviceSince(deviceID string, since time.Time) ([]models.PollResult, error) {
	var results []models.PollResult
	return results, r.db.Where("device_id = ? AND polled_at >= ?", deviceID, since).
		Order("polled_at asc").Find(&results).Error
}

// PruneOlderThan deletes poll results older than the given duration for all devices.
func (r *PollRepository) PruneOlderThan(age time.Duration) error {
	cutoff := time.Now().Add(-age)
	return r.db.Where("polled_at < ?", cutoff).Delete(&models.PollResult{}).Error
}

// SaveAlert persists a status-transition alert event.
func (r *PollRepository) SaveAlert(event *models.AlertEvent) error {
	return r.db.Create(event).Error
}

// FindAlerts returns recent alert events, newest first. If deviceID is empty,
// alerts across all devices are returned.
func (r *PollRepository) FindAlerts(deviceID string, limit int) ([]models.AlertEvent, error) {
	var events []models.AlertEvent
	q := r.db.Order("created_at desc")
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	return events, q.Find(&events).Error
}

// PruneAlertsOlderThan deletes alert events older than the given duration.
func (r *PollRepository) PruneAlertsOlderThan(age time.Duration) error {
	cutoff := time.Now().Add(-age)
	return r.db.Where("created_at < ?", cutoff).Delete(&models.AlertEvent{}).Error
}

func (r *PollRepository) GetSetting(key string) (string, error) {
	var s models.AppSetting
	err := r.db.Where("key = ?", key).First(&s).Error
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

func (r *PollRepository) SetSetting(key, value string) error {
	s := models.AppSetting{Key: key, Value: value, UpdatedAt: time.Now()}
	return r.db.Save(&s).Error
}
