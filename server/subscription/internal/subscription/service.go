package subscription

import "context"

// Service defines the business operations exposed to handlers.
type Service interface {
	Create(context.Context, CreateParams) (Subscription, error)
	GetByID(context.Context, string) (Subscription, error)
	List(context.Context, ListOptions) ([]Subscription, int, error)
	Update(context.Context, UpdateParams) (Subscription, error)
	Delete(context.Context, string) error
	SumByPeriod(context.Context, SumFilter) (int, error)
}

type service struct {
	repo Store
}

// NewService creates a Service backed by the provided repository.
func NewService(repo Store) Service {
	return &service{repo: repo}
}

func (s *service) Create(ctx context.Context, params CreateParams) (Subscription, error) {
	return s.repo.Create(ctx, params)
}

func (s *service) GetByID(ctx context.Context, id string) (Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) List(ctx context.Context, opts ListOptions) ([]Subscription, int, error) {
	return s.repo.List(ctx, opts)
}

func (s *service) Update(ctx context.Context, params UpdateParams) (Subscription, error) {
	return s.repo.Update(ctx, params)
}

func (s *service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) SumByPeriod(ctx context.Context, filter SumFilter) (int, error) {
	return s.repo.SumByPeriod(ctx, filter)
}
