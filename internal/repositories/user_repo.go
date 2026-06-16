package repositories

import (
	"github.com/embrionix/dashboard/internal/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Count() (int64, error) {
	var n int64
	return n, r.db.Model(&models.User{}).Count(&n).Error
}

func (r *UserRepository) FindAll() ([]models.User, error) {
	var users []models.User
	return users, r.db.Order("username asc").Find(&users).Error
}

func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	var u models.User
	if err := r.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(u *models.User) error {
	return r.db.Create(u).Error
}

func (r *UserRepository) Update(u *models.User) error {
	return r.db.Save(u).Error
}

func (r *UserRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}
