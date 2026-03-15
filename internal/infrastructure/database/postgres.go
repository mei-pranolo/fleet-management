package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"fleet-management/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewGormDB creates a GORM DB instance connected to PostgreSQL
func NewGormDB(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm connection: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return db, nil
}

// RunMigrations loads and executes all .sql files from the given folder in
// lexicographic order, so file naming like 0001_, 0002_ controls the sequence.
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations dir %q: %w", migrationsDir, err)
	}

	// Collect and sort SQL files
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	if len(files) == 0 {
		log.Println("[Migration] No SQL migration files found, skipping")
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for migrations: %w", err)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read migration file %q: %w", f, err)
		}

		if _, err := sqlDB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %q: %w", f, err)
		}

		log.Printf("[Migration] Applied: %s", filepath.Base(f))
	}

	return nil
}
