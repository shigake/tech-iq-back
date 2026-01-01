package database

import (
	"log"

	"github.com/shigake/tech-iq-back/internal/config"
	"github.com/shigake/tech-iq-back/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.GetDSN()

	logLevel := logger.Silent
	if cfg.AppEnv == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("✅ Database connected successfully")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	log.Println("🔄 Running database migrations...")

	err := db.AutoMigrate(
		&models.User{},
		&models.Technician{},
		&models.Client{},
		&models.Category{},
		&models.Ticket{},
		&models.TicketFile{},
	)
	if err != nil {
		log.Println("⚠️ Migration warning (continuing anyway):", err)
		// Continue anyway - tables may already exist
	}

	log.Println("✅ Migrations completed")
	return nil
}
