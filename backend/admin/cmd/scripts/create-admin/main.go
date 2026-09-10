// Command create-admin creates an admin-portal account.
//
//	go run ./cmd/scripts/create-admin -email admin@example.com -name "Admin" -password 'secret123' [-role superadmin]
//
// Flags fall back to ADMIN_EMAIL / ADMIN_NAME / ADMIN_PASSWORD / ADMIN_ROLE.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"smartreceipt/backend/common/pkg/database"
	"smartreceipt/backend/common/services/adminuser"

	"smartreceipt/backend/admin/internal/app"
	"smartreceipt/backend/admin/internal/services/adminauth"
)

func main() {
	email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "admin email")
	name := flag.String("name", os.Getenv("ADMIN_NAME"), "display name")
	password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "password (min 8 chars)")
	role := flag.String("role", os.Getenv("ADMIN_ROLE"), "superadmin | admin (default admin)")
	flag.Parse()
	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "create-admin: -email and -password are required")
		os.Exit(2)
	}
	if *name == "" {
		*name = "Admin"
	}
	ctx := context.Background()
	if err := app.ConnectForTooling(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "create-admin: connect: %v\n", err)
		os.Exit(1)
	}
	defer database.Disconnect(context.Background())
	a, err := adminauth.New().CreateAdmin(ctx, *email, *name, *password, *role)
	if err != nil {
		if errors.Is(err, adminuser.ErrEmailTaken) {
			fmt.Fprintln(os.Stderr, "create-admin: an admin with that email already exists")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "create-admin: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("create-admin: OK — %s (%s) id=%s\n", a.Email, a.Role, a.ID.Hex())
}
