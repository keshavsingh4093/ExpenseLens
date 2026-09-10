// Package app boots the admin backend. Run() never builds indexes.
package app

import (
	"context"
	"time"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/database"
	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/logger"

	appconfig "smartreceipt/backend/admin/internal/config"
	"smartreceipt/backend/admin/internal/http/router"
)

// Run boots the service and blocks until shutdown.
func Run() {
	ctx := context.Background()

	if err := appconfig.Load(); err != nil {
		panic("failed to load configuration: " + err.Error())
	}
	logger.Init(appconfig.AppConfig.LogJSON)

	database.ConfigureMongo(database.MongoConfig{
		URI: appconfig.AppConfig.MongoURI, Database: appconfig.AppConfig.MongoDatabase, MaxPoolSize: 50,
	})
	if err := database.Connect(ctx); err != nil {
		logger.Fatal(ctx, "failed to connect to MongoDB", err)
	}
	defer database.Disconnect(context.Background())
	logger.Info(ctx, "mongo connected", "db", appconfig.AppConfig.MongoDatabase)

	if err := config.InitDynamicConfig(ctx); err != nil {
		logger.Warn(ctx, "dynamic config unavailable (using env/defaults)", "err", err)
	}
	if err := appconfig.Reload(); err != nil {
		logger.Warn(ctx, "config reload failed", "err", err)
	}
	if err := appconfig.ValidateProductionSecrets(); err != nil {
		logger.Fatal(ctx, "invalid production configuration", err)
	}
	cfg := appconfig.AppConfig

	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	if missing, err := database.MissingIndexes(checkCtx, AllIndexSets()...); err != nil {
		logger.Warn(ctx, "index check failed", "err", err)
	} else if len(missing) > 0 {
		logger.Warn(ctx, "missing indexes: run `make ensure-indexes`", "missing", missing)
	}
	cancelCheck()

	r := router.InitRouter()
	cors := config.LoadCORS(config.CORSDefaults{
		AllowedOrigins:   []string{cfg.AdminAppURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	srv := gohttp.NewServer(gohttp.ServerConfig{Port: cfg.Port, CORS: cors}, r)
	logger.Info(ctx, "admin api starting", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Run(); err != nil {
		logger.Fatal(ctx, "server error", err)
	}
}
