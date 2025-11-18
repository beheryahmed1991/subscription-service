package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

// Repository handles persistence for subscriptions.
type Store interface {
	Create(context.Context, CreateParams) (Subscription, error)
	GetByID(context.Context, string) (Subscription, error)
	List(context.Context) ([]Subscription, error)
	Update(context.Context, UpdateParams) (Subscription, error)
	Delete(context.Context, string) error
	SumByPeriod(context.Context, SumFilter) (int, error)
}

type Repository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewRepository(db *sql.DB, logger *slog.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (Subscription, error) {
	const query = `
		INSERT INTO subscriptions (service_name, price_rub, user_id, start_month, end_month)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price_rub, user_id, start_month, end_month, created_at, updated_at`

	var sub Subscription
	if err := r.db.QueryRowContext(ctx, query,
		params.ServiceName,
		params.PriceRUB,
		params.UserID,
		params.StartMonth,
		params.EndMonth,
	).Scan(
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
	const query = `
		SELECT id, service_name, price_rub, user_id, start_month, end_month, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	var sub Subscription
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.PriceRUB,
		&sub.UserID,
		&sub.StartMonth,
		&sub.EndMonth,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return Subscription{}, err
		}
		if r.logger != nil {
			r.logger.Error("get subscription failed", "id", id, "error", err)
		}
		return Subscription{}, fmt.Errorf("select subscription: %w", err)
	}

	return sub, nil
}

func (r *Repository) List(ctx context.Context) ([]Subscription, error) {
	const query = `
		SELECT id, service_name, price_rub, user_id, start_month, end_month, created_at, updated_at
		FROM subscriptions
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("list subscriptions query failed", "error", err)
		}
		return nil, fmt.Errorf("list subscriptions: %w", err)
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
			if r.logger != nil {
				r.logger.Error("scan subscription failed", "error", err)
			}
			return nil, fmt.Errorf("scan subscription: %w", err)
		}
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		if r.logger != nil {
			r.logger.Error("iterate subscriptions failed", "error", err)
		}
		return nil, fmt.Errorf("iterate subscriptions: %w", err)
	}

	return subs, nil
}

func (r *Repository) Update(ctx context.Context, params UpdateParams) (Subscription, error) {
	setParts := []string{}
	args := []any{}

	if params.ServiceName != nil {
		args = append(args, *params.ServiceName)
		setParts = append(setParts, fmt.Sprintf("service_name = $%d", len(args)))
	}
	if params.PriceRUB != nil {
		args = append(args, *params.PriceRUB)
		setParts = append(setParts, fmt.Sprintf("price_rub = $%d", len(args)))
	}
	if params.StartMonth != nil {
		args = append(args, *params.StartMonth)
		setParts = append(setParts, fmt.Sprintf("start_month = $%d", len(args)))
	}
	if params.EndMonthSet {
		var end interface{}
		if params.EndMonth != nil {
			end = *params.EndMonth
		}
		args = append(args, end)
		setParts = append(setParts, fmt.Sprintf("end_month = $%d", len(args)))
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, params.ID.String())
	}

	args = append(args, params.ID)
	query := fmt.Sprintf(`
		UPDATE subscriptions
		SET %s, updated_at = now()
		WHERE id = $%d
		RETURNING id, service_name, price_rub, user_id, start_month, end_month, created_at, updated_at`,
		strings.Join(setParts, ", "),
		len(args),
	)

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
		if err == sql.ErrNoRows {
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
	const query = `DELETE FROM subscriptions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("delete subscription failed", "id", id, "error", err)
		}
		return fmt.Errorf("delete subscription: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		if r.logger != nil {
			r.logger.Error("delete rows affected failed", "id", id, "error", err)
		}
		return fmt.Errorf("delete rows affected: %w", err)
	}
	if rows == 0 {
		if r.logger != nil {
			r.logger.Info("subscription not found for delete", "id", id)
		}
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) SumByPeriod(ctx context.Context, filter SumFilter) (int, error) {
	where := []string{}
	args := []any{}

	if filter.StartMonth != nil {
		args = append(args, *filter.StartMonth)
		where = append(where, fmt.Sprintf("start_month >= $%d", len(args)))
	}
	if filter.EndMonth != nil {
		args = append(args, *filter.EndMonth)
		where = append(where, fmt.Sprintf("start_month <= $%d", len(args)))
	}
	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		where = append(where, fmt.Sprintf("user_id = $%d", len(args)))
	}
	if filter.ServiceName != nil {
		args = append(args, strings.ToLower(*filter.ServiceName))
		where = append(where, fmt.Sprintf("lower(service_name) = $%d", len(args)))
	}

	query := `SELECT COALESCE(SUM(price_rub), 0) FROM subscriptions`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	var total sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		if r.logger != nil {
			r.logger.Error("sum subscriptions failed", "error", err)
		}
		return 0, fmt.Errorf("sum subscriptions: %w", err)
	}

	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}
