package data

import (
	"context"
	"time"

	"src/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewMovieRepo,
	NewRatingRepo,
	NewBoxOfficeClient,
)

// Data encapsulates database and cache connections
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
	log *log.Helper
}

// NewData creates Data instance with database and Redis connections
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	l := log.NewHelper(logger)

	// Initialize PostgreSQL connection
	db, err := gorm.Open(postgres.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		l.Errorf("failed to connect to database: %v", err)
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		l.Errorf("failed to get database instance: %v", err)
		return nil, nil, err
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	l.Info("database connected successfully")

	// Initialize Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:         c.Redis.Addr,
		ReadTimeout:  c.Redis.ReadTimeout.AsDuration(),
		WriteTimeout: c.Redis.WriteTimeout.AsDuration(),
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		l.Warnf("failed to connect to redis: %v", err)
		// Redis is optional, continue without it
		rdb = nil
	} else {
		l.Info("redis connected successfully")
	}

	data := &Data{
		db:  db,
		rdb: rdb,
		log: l,
	}

	cleanup := func() {
		l.Info("closing data resources")
		if data.rdb != nil {
			if err := data.rdb.Close(); err != nil {
				l.Errorf("failed to close redis: %v", err)
			}
		}
		if sqlDB != nil {
			if err := sqlDB.Close(); err != nil {
				l.Errorf("failed to close database: %v", err)
			}
		}
	}

	return data, cleanup, nil
}
