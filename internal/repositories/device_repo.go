package repositories

import (
	"github.com/embrionix/dashboard/internal/models"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) FindAll() ([]models.Device, error) {
	var devices []models.Device
	return devices, r.db.Order("name asc").Find(&devices).Error
}

func (r *DeviceRepository) FindByID(id string) (*models.Device, error) {
	var device models.Device
	err := r.db.Where("id = ?", id).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) Create(device *models.Device) error {
	return r.db.Create(device).Error
}

func (r *DeviceRepository) Update(device *models.Device) error {
	return r.db.Save(device).Error
}

func (r *DeviceRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Device{}).Error
}

func (r *DeviceRepository) FindMonitoringEnabled() ([]models.Device, error) {
	var devices []models.Device
	return devices, r.db.Where("monitoring_enabled = ?", true).Find(&devices).Error
}
