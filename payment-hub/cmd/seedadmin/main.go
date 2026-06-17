package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/config"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/database"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./cmd/seedadmin <email> <password>")
		os.Exit(1)
	}
	email := os.Args[1]
	password := os.Args[2]

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	db, err := database.Connect(cfg.DatabaseDSN())
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	hash, err := services.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash: %v\n", err)
		os.Exit(1)
	}

	_, err = db.ExecContext(context.Background(), `
		INSERT INTO admin_users (id, email, password_hash, name, role)
		VALUES (?, ?, ?, ?, 'admin')
		ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash), name = VALUES(name)
	`, security.NewID(), email, hash, "Admin")
	if err != nil {
		fmt.Fprintf(os.Stderr, "insert: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Admin user ready: %s\n", email)
}
