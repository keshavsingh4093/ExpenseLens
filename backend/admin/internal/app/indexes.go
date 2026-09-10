package app

import (
	"context"
	"fmt"

	"smartreceipt/backend/common/pkg/config"
	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/pkg/logger"
	"smartreceipt/backend/common/services/adminaudit"
	"smartreceipt/backend/common/services/adminuser"
	"smartreceipt/backend/common/services/auth"
	"smartreceipt/backend/common/services/systemconfig"

	appconfig "smartreceipt/backend/admin/internal/config"
)

// AllIndexSets lists the indexes the admin backend relies on.
func AllIndexSets() []database.IndexSet {
	return []database.IndexSet{adminuser.Indexes(), auth.TokenIndexes(), config.SystemConfigIndexes(), adminaudit.Indexes()}
}

// EnsureAllIndexes builds the admin backend's indexes (tooling only).
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
	step("adminuser.EnsureIndexes", adminuser.EnsureIndexes)
	step("auth.EnsureIndexes", auth.EnsureIndexes)
	step("systemconfig.EnsureIndexes", systemconfig.EnsureIndexes)
	step("adminaudit.EnsureIndexes", adminaudit.EnsureIndexes)
	if firstErr != nil {
		return fmt.Errorf("ensure-indexes: one or more steps failed; first error: %w", firstErr)
	}
	logger.Info(ctx, "ensure-indexes: all steps completed")
	return nil
}

// ConnectForTooling loads config and connects Mongo for a standalone tool.
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
