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
		return Campaign{}, s.classifyUpdateConflict(ctx, id, datesTouched, newBudget)
	}
	if err != nil {
		return Campaign{}, err
	}
	return updated, nil
}

// classifyUpdateConflict re-reads the row to explain why Update's conditional
// UPDATE matched zero rows: missing/deleted (404), or still present but
// blocked by one of the same conditions the WHERE clause enforces (409, with
// the precise reason). This only differs from the pre-checks in Update in
// the rare case where the row changed concurrently between the two.
func (s *Service) classifyUpdateConflict(ctx context.Context, id int64, datesTouched bool, newBudget *int64) error {
	current, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.NotFound("campaign not found")
	}
	if err != nil {
		return err
	}
	if current.Status == StatusCompleted {
		return httpx.Conflict("completed campaigns cannot be edited")
	}
	if datesTouched && current.Status != StatusPaused {
		return httpx.Conflict("dates can only be changed while the campaign is paused")
	}
	if newBudget != nil && *newBudget < current.Spent {
		return httpx.Conflict("budget cannot be lower than spent")
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
