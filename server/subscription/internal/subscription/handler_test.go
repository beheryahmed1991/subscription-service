package subscription

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type stubStore struct {
	createFn func(context.Context, CreateParams) (Subscription, error)
}

func (s *stubStore) Create(ctx context.Context, params CreateParams) (Subscription, error) {
	if s.createFn != nil {
		return s.createFn(ctx, params)
	}
	return Subscription{}, nil
}

func (s *stubStore) GetByID(context.Context, string) (Subscription, error) {
	return Subscription{}, nil
}

func (s *stubStore) List(context.Context) ([]Subscription, error) {
	return nil, nil
}

func (s *stubStore) Update(context.Context, UpdateParams) (Subscription, error) {
	return Subscription{}, nil
}

func (s *stubStore) Delete(context.Context, string) error {
	return nil
}

func (s *stubStore) SumByPeriod(context.Context, SumFilter) (int, error) {
	return 0, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubStore{
		createFn: func(ctx context.Context, params CreateParams) (Subscription, error) {
			return Subscription{
				ID:          uuid.New(),
				ServiceName: params.ServiceName,
				PriceRUB:    params.PriceRUB,
				UserID:      params.UserID,
				StartMonth:  params.StartMonth,
			}, nil
		},
	}

	h := NewHandler(stub, newTestLogger())

	router := gin.New()
	h.RegisterRoutes(router)

	body := `{
		"service_name":"Netflix",
		"price":499,
		"user_id":"` + uuid.New().String() + `",
		"start_date":"2025-01"
	}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}

func TestHandler_CreateInvalidDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubStore{}
	h := NewHandler(stub, newTestLogger())
	router := gin.New()
	h.RegisterRoutes(router)

	body := `{
		"service_name":"Netflix",
		"price":499,
		"user_id":"` + uuid.New().String() + `",
		"start_date":"invalid-date"
	}`

	req := httptest.NewRequest(http.MethodPost, "/subscriptions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
