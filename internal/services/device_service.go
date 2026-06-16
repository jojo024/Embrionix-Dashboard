package services

import (
	"errors"
	"fmt"

	"github.com/embrionix/dashboard/internal/models"
	"github.com/embrionix/dashboard/internal/repositories"
	"gorm.io/gorm"
)

type DeviceService struct {
	repo *repositories.DeviceRepository
}

func NewDeviceService(repo *repositories.DeviceRepository) *DeviceService {
	return &DeviceService{repo: repo}
}

func (s *DeviceService) ListDevices() ([]models.Device, error) {
	return s.repo.FindAll()
}

func (s *DeviceService) GetDevice(id string) (*models.Device, error) {
	device, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("device not found: %s", id)
		}
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) CreateDevice(device *models.Device) error {
	return s.repo.Create(device)
}

func (s *DeviceService) UpdateDevice(device *models.Device) error {
	_, err := s.GetDevice(device.ID)
	if err != nil {
		return err
	}
	return s.repo.Update(device)
}

func (s *DeviceService) DeleteDevice(id string) error {
	_, err := s.GetDevice(id)
	if err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *DeviceService) ListMonitoringEnabled() ([]models.Device, error) {
	return s.repo.FindMonitoringEnabled()
}
