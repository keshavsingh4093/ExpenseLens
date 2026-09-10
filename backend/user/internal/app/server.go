// Package app boots the user backend. Run() must reach the listener fast:
// no index builds, no blocking migrations (see docs/BACKEND_RESTRUCTURE.md).
package app

import (
	"context"
	"time"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/extraction"
	gohttp "smartreceipt/backend/common/pkg/http"
	"smartreceipt/backend/common/pkg/logger"
	"smartreceipt/backend/common/pkg/storage"

	appconfig "smartreceipt/backend/user/internal/config"
	"smartreceipt/backend/user/internal/http/openapi/files"
	"smartreceipt/backend/user/internal/http/router"
	authsvc "smartreceipt/backend/user/internal/services/auth"
	"smartreceipt/backend/user/internal/services/receipt"
	"smartreceipt/backend/user/internal/services/scheduler"
	"smartreceipt/backend/user/internal/services/warranty"
)

// Run boots the service and blocks until shutdown.
func Run() {
	ctx := context.Background()

	if err := appconfig.Load(); err != nil {
		panic("failed to load configuration: " + err.Error())
	}
	logger.Init(appconfig.AppConfig.LogJSON)

	database.ConfigureMongo(database.MongoConfig{
		URI: appconfig.AppConfig.MongoURI, Database: appconfig.AppConfig.MongoDatabase, MaxPoolSize: 100,
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

	// Check (never build) indexes; warn so ops knows to run ensure-indexes.
	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	if missing, err := database.MissingIndexes(checkCtx, AllIndexSets()...); err != nil {
		logger.Warn(ctx, "index check failed", "err", err)
	} else if len(missing) > 0 {
		logger.Warn(ctx, "missing indexes: run `make ensure-indexes`", "missing", missing)
	}
	cancelCheck()

	// Infrastructure providers.
	store, err := storage.New(storage.Config{
		Provider: cfg.StorageProvider, LocalDir: cfg.LocalStorageDir,
		LocalBaseURL: cfg.APIURL + "/openapi/files", LocalSecret: cfg.JWTSecret,
		CloudinaryURL: cfg.CloudinaryURL,
	})
	if err != nil {
		logger.Fatal(ctx, "storage init failed", err)
	}
	extractor, err := extraction.New(cfg.ExtractionConfig())
	if err != nil {
		logger.Fatal(ctx, "extractor init failed", err)
	}
	pool := extraction.NewPool(cfg.ExtractionWorkers)
	logger.Info(ctx, "providers", "storage", store.Name(), "extractor", extractor.Name(), "workers", cfg.ExtractionWorkers)

	receipt.Configure(receipt.Deps{Store: store, Extractor: extractor, Pool: pool, MaxUploadMB: cfg.MaxUploadMB})
	files.Configure(store)
	warranty.Configure(store)
	receipt.SetHooks(domainHooks())
	authsvc.InitProviders(ctx)
	scheduler.Start()

	r := router.InitRouter()

	cors := config.LoadCORS(config.CORSDefaults{
		AllowedOrigins:   []string{cfg.AppURL},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	srv := gohttp.NewServer(gohttp.ServerConfig{Port: cfg.Port, CORS: cors}, r)
	srv.OnShutdown = append(srv.OnShutdown, scheduler.Stop, receipt.Shutdown)
	logger.Info(ctx, "user api starting", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Run(); err != nil {
		logger.Fatal(ctx, "server error", err)
	}
}
