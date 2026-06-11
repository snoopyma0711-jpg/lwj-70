package database

import (
	"kitchen-trace/internal/config"
	"kitchen-trace/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(cfg *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := models.AutoMigrate(db); err != nil {
		return nil, err
	}
	return db, nil
}
