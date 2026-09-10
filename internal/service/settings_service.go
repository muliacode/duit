package service

import (
	"context"

	"duit/internal/domain"
	"duit/internal/store"
)

type SettingsService struct {
	repo store.SettingsRepository
}

func NewSettingsService(repo store.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) Get(ctx context.Context) (domain.Settings, error) { return s.repo.Get(ctx) }

func (s *SettingsService) Update(ctx context.Context, settings domain.Settings) error {
	return s.repo.Update(ctx, settings)
}
