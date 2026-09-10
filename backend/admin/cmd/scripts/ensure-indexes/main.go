// Command ensure-indexes builds the admin backend's Mongo indexes and exits.
// Run as a deploy step before the server binary is swapped in.
package main

import (
	"context"
	"fmt"
	"os"

	"smartreceipt/backend/common/pkg/database"

	"smartreceipt/backend/admin/internal/app"
)

func main() {
	ctx := context.Background()
	if err := app.ConnectForTooling(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ensure-indexes: failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer database.Disconnect(context.Background())
	if err := app.EnsureAllIndexes(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ensure-indexes: FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ensure-indexes: OK — all indexes are in place")
}
