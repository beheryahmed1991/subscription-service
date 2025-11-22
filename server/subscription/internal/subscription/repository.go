package subscription

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	goqu "github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
)

// Store describes the contract for subscription persistence.
type Store interface {
	Create(context.Context, CreateParams) (Subscription, error)
	GetByID(context.Context, string) (Subscription, error)
	List(context.Context, ListOptions) ([]Subscription, int, error)
	Update(context.Context, UpdateParams) (Subscription, error)
	Delete(context.Context, string) error
	SumByPeriod(context.Context, SumFilter) (int, error)
}

// ListOptions controls pagination for List.
type ListOptions struct {
	Limit  int
	Offset int
}

// Repository is the goqu-backed implementation of Store.
type Repository struct {
	db      *sql.DB
	logger  *slog.Logger
	builder *goqu.Database
}

// NewRepository wires the DB and logger into a Repository.
func NewRepository(db *sql.DB, logger *slog.Logger) *Repository {
	return &Repository{
		db:      db,
		logger:  logger,
		builder: goqu.New("postgres", db),
	}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (Subscription, error) {
	stmt := r.builder.Insert("subscriptions").Rows(goqu.Record{
		"service_name": params.ServiceName,
		"price_rub":    params.PriceRUB,
		"user_id":      params.UserID,
		"start_month":  params.StartMonth,
		"end_month":    params.EndMonth,
	}).Returning(
		"id", "service_name", "price_rub", "user_id", "start_month", "end_month", "created_at", "updated_at",
	)

	query, args, err := stmt.ToSQL()
	if err != nil {
		return Subscription{}, fmt.Errorf("build insert subscription: %w", err)
	}

	var sub Subscription
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.PriceRUB,
		&sub.UserID,
		&sub.StartMonth,
		&sub.EndMonth,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	); err != nil {
		if r.logger != nil {
			r.logger.Error("insert subscription failed", "error", err)
		}
		return Subscription{}, fmt.Errorf("insert subscription: %w", err)
	}

	return sub, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (Subscription, error) {
	ds := r.builder.From("subscriptions").Select(
		"id", "service_name", "price_rub", "user_id", "start_month", "end_month", "created_at", "updated_at",
	).Where(goqu.C("id").Eq(id))

	query, args, err := ds.ToSQL()
	if err != nil {
		return Subscription{}, fmt.Errorf("build get subscription: %w", err)
	}

	var sub Subscription
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.PriceRUB,
		&sub.UserID,
		&sub.StartMonth,
		&sub.EndMonth,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, err
		}
		if r.logger != nil {
			r.logger.Error("get subscription failed", "id", id, "error", err)
		}
		return Subscription{}, fmt.Errorf("select subscription: %w", err)
	}

	return sub, nil
}

func (r *Repository) List(ctx context.Context, opts ListOptions) ([]Subscription, int, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	listDS := r.builder.From("subscriptions").Select(
		"id", "service_name", "price_rub", "user_id", "start_month", "end_month", "created_at", "updated_at",
	).Order(goqu.I("created_at").Desc()).Limit(uint(limit)).Offset(uint(offset))

	query, args, err := listDS.ToSQL()
	if err != nil {
		return nil, 0, fmt.Errorf("build list subscriptions: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("list subscriptions query failed", "error", err)
		}
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.PriceRUB,
			&sub.UserID,
			&sub.StartMonth,
			&sub.EndMonth,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	countDS := r.builder.From("subscriptions").Select(goqu.COUNT("*"))
	countQuery, countArgs, err := countDS.ToSQL()
	if err != nil {
		return nil, 0, fmt.Errorf("build count subscriptions: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	return subs, total, nil
}

func (r *Repository) Update(ctx context.Context, params UpdateParams) (Subscription, error) {
	updates := goqu.Record{}

	if params.ServiceName != nil {
		updates["service_name"] = *params.ServiceName
	}
	if params.PriceRUB != nil {
		updates["price_rub"] = *params.PriceRUB
	}
	if params.StartMonth != nil {
		updates["start_month"] = *params.StartMonth
	}
	if params.EndMonthSet {
		if params.EndMonth != nil {
			updates["end_month"] = *params.EndMonth
		} else {
			updates["end_month"] = nil
		}
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, params.ID.String())
	}

	updates["updated_at"] = goqu.L("now()")

	ds := r.builder.Update("subscriptions").
		Set(updates).
		Where(goqu.C("id").Eq(params.ID)).
		Returning("id", "service_name", "price_rub", "user_id", "start_month", "end_month", "created_at", "updated_at")

	query, args, err := ds.ToSQL()
	if err != nil {
		return Subscription{}, fmt.Errorf("build update subscription: %w", err)
	}

	var sub Subscription
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.PriceRUB,
		&sub.UserID,
		&sub.StartMonth,
		&sub.EndMonth,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Subscription{}, err
		}
		if r.logger != nil {
			r.logger.Error("update subscription failed", "id", params.ID, "error", err)
		}
		return Subscription{}, fmt.Errorf("update subscription: %w", err)
	}

	return sub, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	ds := r.builder.Delete("subscriptions").Where(goqu.C("id").Eq(id))
	query, args, err := ds.ToSQL()
	if err != nil {
		return fmt.Errorf("build delete subscription: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("delete subscription failed", "id", id, "error", err)
		}
		return fmt.Errorf("delete subscription: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		if r.logger != nil {
			r.logger.Info("subscription not found for delete", "id", id)
		}
		return sql.ErrNoRows
	}

	return nil
}

const sumByPeriodSQL = `
WITH ranges AS (
    SELECT
        s.price_rub,
        GREATEST(s.start_month, COALESCE($1::date, s.start_month)) AS eff_start,
        LEAST(
            COALESCE(s.end_month, COALESCE($2::date, CURRENT_DATE)),
            COALESCE($2::date, COALESCE(s.end_month, CURRENT_DATE))
        ) AS eff_end
    FROM subscriptions s
    WHERE ($3::uuid IS NULL OR s.user_id = $3::uuid)
      AND ($4::text IS NULL OR LOWER(s.service_name) = LOWER($4::text))
      AND s.start_month <= COALESCE($2::date, COALESCE(s.end_month, CURRENT_DATE))
      AND COALESCE(s.end_month, COALESCE($2::date, CURRENT_DATE)) >= COALESCE($1::date, s.start_month)
)
SELECT COALESCE(SUM(
    price_rub *
    (
        (DATE_PART('year', eff_end) - DATE_PART('year', eff_start)) * 12 +
        (DATE_PART('month', eff_end) - DATE_PART('month', eff_start)) + 1
    )
), 0)
FROM ranges
WHERE eff_end >= eff_start;
`

func (r *Repository) SumByPeriod(ctx context.Context, filter SumFilter) (int, error) {
	var (
		start interface{}
		end   interface{}
		user  interface{}
		name  interface{}
	)

	if filter.StartMonth != nil {
		start = normalizeMonth(*filter.StartMonth)
	}
	if filter.EndMonth != nil {
		end = normalizeMonth(*filter.EndMonth)
	}
	if filter.UserID != nil {
		user = *filter.UserID
	}
	if filter.ServiceName != nil {
		name = strings.TrimSpace(*filter.ServiceName)
		if name == "" {
			name = nil
		}
	}

	var total sql.NullInt64
	if err := r.db.QueryRowContext(ctx, sumByPeriodSQL, start, end, user, name).Scan(&total); err != nil {
		return 0, fmt.Errorf("sum subscriptions: %w", err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

func monthsBetween(start, end time.Time) int {
	start = normalizeMonth(start)
	end = normalizeMonth(end)
	if end.Before(start) {
		return 0
	}
	years := end.Year() - start.Year()
	months := int(end.Month()) - int(start.Month())
	return years*12 + months + 1
}

func clampRange(subStart time.Time, subEnd sql.NullTime, periodStart, periodEnd *time.Time) (time.Time, time.Time, bool) {
	start := normalizeMonth(subStart)

	var subEndNorm *time.Time
	if subEnd.Valid {
		t := normalizeMonth(subEnd.Time)
		subEndNorm = &t
	}

	if periodStart != nil {
		ps := normalizeMonth(*periodStart)
		if ps.After(start) {
			start = ps
		}
	}

	var end time.Time
	switch {
	case subEndNorm != nil && periodEnd != nil:
		pe := normalizeMonth(*periodEnd)
		if pe.Before(*subEndNorm) {
			end = pe
		} else {
			end = *subEndNorm
		}
	case subEndNorm != nil && periodEnd == nil:
		end = *subEndNorm
	case subEndNorm == nil && periodEnd != nil:
		end = normalizeMonth(*periodEnd)
	default:
		end = normalizeMonth(time.Now().UTC())
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
