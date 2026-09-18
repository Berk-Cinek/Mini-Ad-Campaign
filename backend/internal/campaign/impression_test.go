package campaign_test

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Berk-Cinek/Mini-Ad-Campaign/backend/internal/campaign"
)

// TestConcurrentImpressions_NeverOverspend proves the Definition of Done
// requirement: under heavy concurrent load against a single campaign, spent
// never exceeds budget. It runs against the real Postgres started by
// docker compose, per CLAUDE.md's testing approach (no mocks).
func TestConcurrentImpressions_NeverOverspend(t *testing.T) {
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

	const budget = 50
	const concurrency = 300

	created, err := svc.Create(ctx, campaign.CreateInput{
		Title:     "concurrency test",
		Budget:    json.Number(strconv.Itoa(budget)),
		StartDate: time.Now().Add(-time.Hour),
		EndDate:   time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("creating test campaign: %v", err)
	}
	t.Cleanup(func() {
		_ = svc.Delete(context.Background(), created.ID)
	})

	var successes int64
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			if _, err := svc.RecordImpression(ctx, created.ID); err == nil {
				atomic.AddInt64(&successes, 1)
			}
		}()
	}
	wg.Wait()

	if successes != budget {
		t.Fatalf("expected exactly %d successful impressions, got %d", budget, successes)
	}

	final, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("fetching final campaign state: %v", err)
	}
	if final.Spent != budget {
		t.Fatalf("expected spent == %d, got %d", budget, final.Spent)
	}
	if final.Spent > final.Budget {
		t.Fatalf("budget went negative: spent=%d budget=%d", final.Spent, final.Budget)
	}
	if final.Status != campaign.StatusPaused {
		t.Fatalf("expected status %q after exhausting budget, got %q", campaign.StatusPaused, final.Status)
	}
}
