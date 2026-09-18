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

func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) (Campaign, error) {
	current, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, httpx.NotFound("campaign not found")
	}
	if err != nil {
		return Campaign{}, err
	}

	if current.Status == StatusCompleted {
		return Campaign{}, httpx.Conflict("completed campaigns cannot be edited")
	}

	datesTouched := in.StartDate != nil || in.EndDate != nil
	if datesTouched && current.Status != StatusPaused {
		return Campaign{}, httpx.Conflict("dates can only be changed while the campaign is paused")
	}

	var newTitle *string
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return Campaign{}, httpx.Invalid("title is required")
		}
		if utf8.RuneCountInString(title) > 200 {
			return Campaign{}, httpx.Invalid("title must be at most 200 characters")
		}
		newTitle = &title
	}

	var newBudget *int64
	if in.Budget != nil {
		budget, err := strconv.ParseInt(string(*in.Budget), 10, 64)
		if err != nil || budget < 1 {
			return Campaign{}, httpx.Invalid("budget must be a whole number >= 1")
		}
		// Comparing against the campaign's current state (spent), not a
		// property of the value alone, so this is a conflict (409) like the
		// completed/dates-while-active checks above, not an input error (400).
		if budget < current.Spent {
			return Campaign{}, httpx.Conflict("budget cannot be lower than spent")
		}
		newBudget = &budget
	}

	var newStart, newEnd *time.Time
	if datesTouched {
		effectiveStart := current.StartDate
		if in.StartDate != nil {
			if in.StartDate.IsZero() {
				return Campaign{}, httpx.Invalid("start_date cannot be empty")
			}
			effectiveStart = in.StartDate.UTC()
			newStart = &effectiveStart
		}
		effectiveEnd := current.EndDate
		if in.EndDate != nil {
			if in.EndDate.IsZero() {
				return Campaign{}, httpx.Invalid("end_date cannot be empty")
			}
			effectiveEnd = in.EndDate.UTC()
			newEnd = &effectiveEnd
		}
		if !effectiveEnd.After(effectiveStart) {
			return Campaign{}, httpx.Invalid("end_date must be after start_date")
		}
		if !effectiveEnd.After(time.Now()) {
			return Campaign{}, httpx.Invalid("end_date must be in the future")
		}
	}

	updated, err := s.repo.Update(ctx, id, newTitle, newBudget, newStart, newEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, s.classifyUpdateConflict(ctx, id,
			conflictCheck{func(c Campaign) bool { return c.Status == StatusCompleted }, "completed campaigns cannot be edited"},
			conflictCheck{func(c Campaign) bool { return datesTouched && c.Status != StatusPaused }, "dates can only be changed while the campaign is paused"},
			conflictCheck{func(c Campaign) bool { return newBudget != nil && *newBudget < c.Spent }, "budget cannot be lower than spent"},
		)
	}
	if err != nil {
		return Campaign{}, err
	}
	return updated, nil
}

type conflictCheck struct {
	failed  func(Campaign) bool
	message string
}

// classifyUpdateConflict explains why a conditional UPDATE (Update's PATCH,
// or the impression endpoint's Deduct) matched zero rows: re-reads the row
// and returns 404 if it's missing/deleted, otherwise runs the caller-supplied
// checks in order and returns the first one whose condition still holds as a
// 409 — falling back to a generic conflict message if none match (only
// reachable if the row changed again between the original write attempt and
// this re-read).
func (s *Service) classifyUpdateConflict(ctx context.Context, id int64, checks ...conflictCheck) error {
	current, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.NotFound("campaign not found")
	}
	if err != nil {
		return err
	}
	for _, check := range checks {
		if check.failed(current) {
			return httpx.Conflict(check.message)
		}
	}
	return httpx.Conflict("campaign was modified concurrently, please retry")
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.repo.SoftDelete(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.NotFound("campaign not found")
	}
	return err
}

func (s *Service) Pause(ctx context.Context, id int64) (Campaign, error) {
	c, err := s.repo.Pause(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, s.classifyUpdateConflict(ctx, id,
			conflictCheck{func(c Campaign) bool { return c.Status != StatusActive }, "campaign is not active"},
		)
	}
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}

func (s *Service) Resume(ctx context.Context, id int64) (Campaign, error) {
	c, err := s.repo.Resume(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, s.classifyUpdateConflict(ctx, id,
			conflictCheck{func(c Campaign) bool { return c.Status != StatusPaused }, "campaign is not paused"},
			conflictCheck{func(c Campaign) bool { return c.Spent >= c.Budget }, "no budget left"},
			conflictCheck{func(c Campaign) bool { return !c.EndDate.After(time.Now()) }, "end date has passed"},
		)
	}
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}

func (s *Service) End(ctx context.Context, id int64) (Campaign, error) {
	c, err := s.repo.End(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Campaign{}, s.classifyUpdateConflict(ctx, id,
			conflictCheck{func(c Campaign) bool { return c.Status == StatusCompleted }, "campaign is already completed"},
		)
	}
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}

func (s *Service) Stats(ctx context.Context, id int64) (Stats, error) {
	c, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Stats{}, httpx.NotFound("campaign not found")
	}
	if err != nil {
		return Stats{}, err
	}
	return c.Stats(), nil
}

func (s *Service) RecordImpression(ctx context.Context, id int64) (Campaign, error) {
	c, err := s.repo.Deduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		now := time.Now()
		return Campaign{}, s.classifyUpdateConflict(ctx, id,
			conflictCheck{func(c Campaign) bool { return c.Status != StatusActive }, "campaign is not active"},
			conflictCheck{func(c Campaign) bool { return now.Before(c.StartDate) }, "campaign has not started yet"},
			conflictCheck{func(c Campaign) bool { return !now.Before(c.EndDate) }, "campaign has ended"},
			conflictCheck{func(c Campaign) bool { return c.Spent >= c.Budget }, "campaign budget is exhausted"},
		)
	}
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}
