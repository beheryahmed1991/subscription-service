package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db, nil)

	userID := uuid.New()
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "service_name", "price_rub", "user_id", "start_month", "end_month", "created_at", "updated_at",
	}).AddRow(uuid.New(), "Netflix", 499, userID, start, nil, now, now)

	mock.ExpectQuery("INSERT INTO subscriptions").
		WithArgs("Netflix", 499, userID, start, (*time.Time)(nil)).
		WillReturnRows(rows)

	sub, err := repo.Create(context.Background(), CreateParams{
		ServiceName: "Netflix",
		PriceRUB:    499,
		UserID:      userID,
		StartMonth:  start,
		EndMonth:    nil,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if sub.ServiceName != "Netflix" || sub.PriceRUB != 499 {
		t.Fatalf("unexpected subscription: %+v", sub)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestRepository_CreateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	repo := NewRepository(db, nil)

	userID := uuid.New()
	start := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("INSERT INTO subscriptions").
		WithArgs("Netflix", 499, userID, start, (*time.Time)(nil)).
		WillReturnError(context.DeadlineExceeded)

	if _, err := repo.Create(context.Background(), CreateParams{
		ServiceName: "Netflix",
		PriceRUB:    499,
		UserID:      userID,
		StartMonth:  start,
	}); err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
