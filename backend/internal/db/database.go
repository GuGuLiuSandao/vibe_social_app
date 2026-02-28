package db

import (
	"fmt"
	"social_app/internal/config"
	app_logger "social_app/internal/logger"
	"social_app/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
)

var DB *gorm.DB

const whitelistMinUID = 10000000
const userUIDSequenceName = "user_uid_seq"

func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   gorm_logger.Default.LogMode(gorm_logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	app_logger.Info("Database connected successfully")

	err = autoMigrate()
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

func autoMigrate() error {
	app_logger.DBWrite("MIGRATE", "all_tables", "Starting database migration")

	err := DB.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.Conversation{},
		&models.ConversationParticipant{},
		&models.Relation{},
	)

	if err != nil {
		app_logger.Error("Database migration failed: %v", err)
		return err
	}

	if err := ensureUserUIDSequence(); err != nil {
		app_logger.Error("Failed to ensure user ID sequence: %v", err)
		return err
	}

	app_logger.DBWrite("MIGRATE", "all_tables", "Database migration completed successfully")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

func ensureUserUIDSequence() error {
	// Create sequence if not exists
	if err := DB.Exec(fmt.Sprintf("CREATE SEQUENCE IF NOT EXISTS %s START 20000001", userUIDSequenceName)).Error; err != nil {
		return err
	}

	// Sync sequence with max uid in non-whitelist range (if any)
	// We assume normal users start from 20000000 or whatever sequence defines.
	// But whitelist users (20000001+) might conflict if sequence overlaps.
	// User said "id=20000005, uid=20000005".
	// Let's just ensure sequence is at least MAX(uid) + 1.
	return DB.Exec(fmt.Sprintf(`
		SELECT setval(
			'%s',
			GREATEST((SELECT COALESCE(MAX(uid), 20000000) FROM users), 20000000)
		)
	`, userUIDSequenceName)).Error
}
