// Command add-admin adds a college's first admin, once (ADR 0026):
//
//	go run ./cmd/add-admin -email office@college.edu -name "Nikhil Bhat"
//
// It reads the same environment as the API (DATABASE_URL and the rest).
// The admin then signs in with Google using that email; there is no
// password. Later admins are granted inside LINKS.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/AbhishekBalija/Links/server/internal/auth"
	"github.com/AbhishekBalija/Links/server/internal/mailer"
	"github.com/AbhishekBalija/Links/server/migrations"
	"github.com/AbhishekBalija/Links/server/pkg/config"
	"github.com/AbhishekBalija/Links/server/pkg/db"
	"gorm.io/gorm"
)

func main() {
	email := flag.String("email", "", "the admin's email; they sign in with Google using it")
	name := flag.String("name", "", "the admin's full name")
	flag.Parse()
	if err := run(*email, *name); err != nil {
		fmt.Fprintln(os.Stderr, "add-admin:", err)
		os.Exit(1)
	}
	fmt.Printf("Added %s as the first admin. They can now sign in with Google using %s.\n", *name, *email)
}

func run(email, name string) error {
	if email == "" || name == "" {
		return errors.New("both -email and -name are needed")
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	database, err := db.New(cfg)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// The admin's account needs the current tables.
	if err := database.Migrate(ctx, migrations.FS); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return addAdmin(ctx, database.GORM(), email, name)
}

// addAdmin adds the first admin through the auth service, so it follows the
// same rules and audit as everything else.
func addAdmin(ctx context.Context, gdb *gorm.DB, email, name string) error {
	service := auth.NewAuthService(
		auth.NewGormUserRepository(gdb),
		auth.NewGormRefreshTokenRepository(gdb),
		auth.NewGormActivationTokenRepository(gdb),
		auth.NewGormAuthUnitOfWork(gdb),
		auth.TokenConfig{},
		auth.NewArgon2PasswordHasher(),
		mailer.NoopMailer{},
		"",
		auth.DefaultCodeSettings(),
		nil,
	)
	_, err := service.AddFirstAdmin(ctx, email, name)
	return err
}
