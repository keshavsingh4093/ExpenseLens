// Command ensure-indexes builds every Mongo index for the user backend and
// seeds the system categories, then exits. Run it as a deploy step BEFORE
// the server binary is swapped in; the server never builds indexes itself.
//
// Exit codes: 0 all indexes ensured; 1 connection or index step failed.
package main

import (
	"context"
	"fmt"
	"os"

	"smartreceipt/backend/common/pkg/database"

	"smartreceipt/backend/user/internal/app"
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
