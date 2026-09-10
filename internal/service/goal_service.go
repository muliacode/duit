package service

import (
	"context"

	"duit/internal/domain"
	"duit/internal/store"
)

type GoalService struct {
	repo store.GoalRepository
}

func NewGoalService(repo store.GoalRepository) *GoalService { return &GoalService{repo: repo} }

func (s *GoalService) List(ctx context.Context) ([]domain.Goal, error) { return s.repo.List(ctx) }

func (s *GoalService) Create(ctx context.Context, g domain.Goal) (domain.Goal, error) {
	if err := g.Validate(); err != nil {
		return domain.Goal{}, err
	}
	return s.repo.Create(ctx, g)
}

func (s *GoalService) Update(ctx context.Context, g domain.Goal) error {
	if err := g.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, g)
}

func (s *GoalService) Delete(ctx context.Context, id string) error { return s.repo.Delete(ctx, id) }
