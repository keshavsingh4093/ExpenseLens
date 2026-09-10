package app

import (
	"context"
	"fmt"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/logger"
	"smartreceipt/backend/common/services/auth"
	"smartreceipt/backend/common/services/category"
	"smartreceipt/backend/common/services/receipt"
	"smartreceipt/backend/common/services/user"

	appconfig "smartreceipt/backend/user/internal/config"
)

// AllIndexSets lists every index the user backend relies on. Run() only
// checks these; cmd/scripts/ensure-indexes builds them.
func AllIndexSets() []database.IndexSet {
	return []database.IndexSet{
		user.Indexes(), auth.TokenIndexes(), category.Indexes(), receipt.Indexes(), config.SystemConfigIndexes(),
	}
}

// EnsureAllIndexes builds every index and seeds the system categories. It is
// the single place schema setup happens and must never run on the boot path.
func EnsureAllIndexes(ctx context.Context) error {
	var firstErr error
	step := func(name string, fn func(context.Context) error) {
		logger.Info(ctx, "ensure-indexes: "+name)
		if err := fn(ctx); err != nil {
			logger.Error(ctx, "ensure-indexes: "+name+" FAILED", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	step("user.EnsureIndexes", user.EnsureIndexes)
	step("auth.EnsureIndexes", auth.EnsureIndexes)
	step("category.EnsureIndexes", category.EnsureIndexes)
	step("category.Seed", category.Seed)
	step("receipt.EnsureIndexes", receipt.EnsureIndexes)
	step("systemconfig.EnsureIndexes", func(ctx context.Context) error {
		return database.EnsureIndexes(ctx, config.SystemConfigIndexes())
	})
	if firstErr != nil {
		return fmt.Errorf("ensure-indexes: one or more steps failed; first error: %w", firstErr)
	}
	logger.Info(ctx, "ensure-indexes: all steps completed")
	return nil
}

// ConnectForTooling loads config and connects Mongo for a standalone tool.
// The caller disconnects.
func ConnectForTooling(ctx context.Context) error {
	if err := appconfig.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger.Init(appconfig.AppConfig.LogJSON)
	database.ConfigureMongo(database.MongoConfig{
		URI: appconfig.AppConfig.MongoURI, Database: appconfig.AppConfig.MongoDatabase, MaxPoolSize: 20,
	})
	if err := database.Connect(ctx); err != nil {
		return fmt.Errorf("connect mongo: %w", err)
	}
	if err := config.InitDynamicConfig(ctx); err != nil {
		logger.Warn(ctx, "dynamic config unavailable (using env/defaults)", "err", err)
	}
	return appconfig.Reload()
}
