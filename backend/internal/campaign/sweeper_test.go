package campaign_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Berk-Cinek/Mini-Ad-Campaign/backend/internal/campaign"
)

// TestSweeper_CompletesPastEndDate proves the background sweep (CLAUDE.md:
// "a background job runs every minute and marks campaigns past end_date as
// completed") actually flips status, and leaves campaigns still within
// their window untouched. It's a catch-up/cosmetic job, not a correctness
// mechanism, so this exercises the repository method the sweeper calls
// directly rather than waiting out a real ticker interval.
func TestSweeper_CompletesPastEndDate(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; run against the Postgres started by docker compose")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	repo := campaign.NewRepository(pool)
	svc := campaign.NewService(repo)

	// Created with a valid (future) end_date, since Create rejects a past
	// one, then backdated directly in SQL to simulate a campaign whose
	// window has since elapsed.
	expired, err := svc.Create(ctx, campaign.CreateInput{
		Title:     "sweeper test - expired",
		Budget:    json.Number("100"),
		StartDate: time.Now().Add(-2 * time.Hour),
		EndDate:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("creating expired test campaign: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), expired.ID) })

	if _, err := pool.Exec(ctx, `UPDATE campaigns SET end_date = $1 WHERE id = $2`,
		time.Now().Add(-time.Minute), expired.ID); err != nil {
		t.Fatalf("backdating end_date: %v", err)
	}

	current, err := svc.Create(ctx, campaign.CreateInput{
		Title:     "sweeper test - current",
		Budget:    json.Number("100"),
		StartDate: time.Now().Add(-time.Hour),
		EndDate:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("creating current test campaign: %v", err)
	}
	t.Cleanup(func() { _ = svc.Delete(context.Background(), current.ID) })

	if _, err := repo.CompletePastEndDate(ctx); err != nil {
		t.Fatalf("sweeping: %v", err)
	}

	swept, err := svc.Get(ctx, expired.ID)
	if err != nil {
		t.Fatalf("fetching expired campaign: %v", err)
	}
	if swept.Status != campaign.StatusCompleted {
		t.Fatalf("expected expired campaign to be completed, got %q", swept.Status)
	}
	if !swept.UpdatedAt.After(expired.UpdatedAt) {
		t.Fatalf("expected updated_at to advance after sweep")
	}

	untouched, err := svc.Get(ctx, current.ID)
	if err != nil {
		t.Fatalf("fetching current campaign: %v", err)
	}
	if untouched.Status != campaign.StatusActive {
		t.Fatalf("expected current campaign to remain active, got %q", untouched.Status)
	}
}
