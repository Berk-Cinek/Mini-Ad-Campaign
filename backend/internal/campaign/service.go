package campaign

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Berk-Cinek/Mini-Ad-Campaign/backend/internal/httpx"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Campaign, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return Campaign{}, httpx.Invalid("title is required")
	}
	if utf8.RuneCountInString(title) > 200 {
		return Campaign{}, httpx.Invalid("title must be at most 200 characters")
	}

	budget, err := strconv.ParseInt(string(in.Budget), 10, 64)
	if err != nil || budget < 1 {
		return Campaign{}, httpx.Invalid("budget must be a whole number >= 1")
	}

	// A missing/null JSON date decodes to time.Time's zero value (0001-01-01),
	// not an error, so it must be checked explicitly — otherwise an omitted
	// start_date passes both After() checks below (zero time is before any
	// real end_date, and any real end_date is after "now").
	if in.StartDate.IsZero() {
		return Campaign{}, httpx.Invalid("start_date is required")
	}
	if in.EndDate.IsZero() {
		return Campaign{}, httpx.Invalid("end_date is required")
	}

	if !in.EndDate.After(in.StartDate) {
		return Campaign{}, httpx.Invalid("end_date must be after start_date")
	}
	if !in.EndDate.After(time.Now()) {
		return Campaign{}, httpx.Invalid("end_date must be in the future")
	}

	return s.repo.Create(ctx, Campaign{
		Title:     title,
		Budget:    budget,
		StartDate: in.StartDate.UTC(),
		EndDate:   in.EndDate.UTC(),
	})
}

func (s *Service) List(ctx context.Context) ([]Campaign, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (Campaign, error) {
	c, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, httpx.NotFound("campaign not found")
	}
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}
