package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"monitoring-energy-service/internal/infrastructure/adapters/repositories"
	"monitoring-energy-service/internal/infrastructure/conf"

	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func SetupDatabasePsql() (*gorm.DB, error) {
	dbSetting, err := conf.LoadDBSettings()
	if err != nil {
		return nil, err
	}

	DBuri := conf.DBUriPsql(dbSetting)
	db, err := gorm.Open(postgres.Open(DBuri), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("unable to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(dbSetting.DbMaxIdleConns)
	sqlDB.SetMaxOpenConns(dbSetting.DbMaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Minute * time.Duration(dbSetting.DbConnMaxLifetime))
	sqlDB.SetConnMaxIdleTime(time.Minute * time.Duration(dbSetting.DbConnMaxIdleTime))

	return db, nil
}

func PerformMigrations(db *gorm.DB) error {
	possiblePaths := []string{
		"./migrations",
		"../migrations",
		"../../migrations",
		"../../../migrations",
		"../../../../migrations",
	}

	if rootDir, err := findProjectRoot(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(rootDir, "migrations"))
	}

	var migrationsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			migrationsDir = path
			break
		}
	}

	if migrationsDir == "" {
		slog.Warn("no migrations directory found, skipping migrations")
		return nil
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("error accessing underlying database connection: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error setting dialect for goose: %w", err)
	}

	slog.Info("running database migrations", "directory", migrationsDir)
	if err := goose.Up(sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("error running migrations: %w in directory %s", err, migrationsDir)
	}

	return nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found, cannot determine project root")
		}
		dir = parent
	}
}

func SeedDB(db *gorm.DB, directory string) error {
	entitiesFile := []struct {
		name   string
		entity interface{}
	}{
		{"energy_plants", &repositories.EnergyPlantsModel{}},
	}

	for _, item := range entitiesFile {
		name := item.name
		entity := item.entity
		var count int64
		if err := db.Model(entity).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to count %s records: %w", name, err)
		}
		if count == 0 {
			slog.Info("seeding model", "model", name)
			err := seedData(db, directory, name)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func seedData(db *gorm.DB, directory, name string) error {
	sqlBytes, err := os.ReadFile(fmt.Sprintf("%s/%s.sql", directory, name))
	if err != nil {
		return err
	}

	sql := string(sqlBytes)
	if err := db.Exec(sql).Error; err != nil {
		return err
	}

	return nil
}
