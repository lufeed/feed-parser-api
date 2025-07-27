package database

import (
	"fmt"
	"github.com/lufeed/feed-parser-api/internal/config"
	"github.com/lufeed/feed-parser-api/internal/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"sync"
)

var (
	db   *gorm.DB
	once sync.Once
)

func Initialize(cfg *config.AppConfig) error {
	var err error
	once.Do(func() {
		err = initializePostgreSQL(cfg)
		if err != nil {
			logger.GetSugaredLogger().Error("Failed to initialize PostgreSQL database", zap.Error(err))
			return
		}
	})
	return err
}

func initializePostgreSQL(cfg *config.AppConfig) error {
	var err error
	databaseConfig := cfg.Database
	db, err = gorm.Open(postgres.Open(getPostgreSQLConnectionDSN(databaseConfig)), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent),
	})
	if err != nil {
		logger.GetLogger().With(zap.Error(err)).Fatal("Failed to connect to PostgreSQL")
		return err
	}

	logger.GetLogger().With(zap.Error(err)).Info("Connected to database")
	return nil
}

func getPostgreSQLConnectionDSN(conf config.DatabaseConfig) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s  sslmode=disable TimeZone=UTC search_path=%s",
		conf.Host,
		conf.Port,
		conf.User,
		conf.Pass,
		conf.Dbname,
		conf.Schema,
	)
}
