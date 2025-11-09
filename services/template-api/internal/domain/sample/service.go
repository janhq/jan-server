package sample

import (
	"context"

	"github.com/rs/zerolog"
)

// Service describes the business logic surface for sample operations.
type Service interface {
	GetSample(ctx context.Context) (Sample, error)
}

type service struct {
	repo Repository
	log  zerolog.Logger
}

// NewService wires the sample service with its repository.
func NewService(repo Repository, log zerolog.Logger) Service {
	return &service{
		repo: repo,
		log:  log.With().Str("component", "sample-service").Logger(),
	}
}

func (s *service) GetSample(ctx context.Context) (Sample, error) {
	result, err := s.repo.FetchLatest(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("fetch latest sample")
		return Sample{}, err
	}
	return result, nil
}
