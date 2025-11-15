package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"

	"github.com/beheryahmed1991/subscription-service.git/migrations"
)

// Up runs embedded Goose migrations.
func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations.Files)
	goose.SetVerbose(false)

	if err := goose.RunContext(ctx, "up", db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
