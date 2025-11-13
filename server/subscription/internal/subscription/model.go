package subscription

import (
	"time"

	"github.com/google/uuid"
)

// Subscription mirrors the database schema for the subscriptions table.
type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	PriceRUB    int        `json:"price_rub"`
	UserID      uuid.UUID  `json:"user_id"`
	StartMonth  time.Time  `json:"start_month"`
	EndMonth    *time.Time `json:"end_month,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateParams represents validated data needed to insert a subscription.
type CreateParams struct {
	ServiceName string
	PriceRUB    int
	UserID      uuid.UUID
	StartMonth  time.Time
	EndMonth    *time.Time
}

// UpdateParams carries mutable fields for an existing subscription.
type UpdateParams struct {
	ID          uuid.UUID
	ServiceName *string
	PriceRUB    *int
	StartMonth  *time.Time
	EndMonth    *time.Time
	EndMonthSet bool
}

// SumFilter describes filters for aggregation queries.
type SumFilter struct {
	StartMonth  *time.Time
	EndMonth    *time.Time
	UserID      *uuid.UUID
	ServiceName *string
}
