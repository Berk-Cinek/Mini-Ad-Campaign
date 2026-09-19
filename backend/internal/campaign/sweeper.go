package campaign

import (
	"context"
	"log"
	"time"
)

// Sweeper periodically marks campaigns past their end_date as completed.
// It's a cosmetic catch-up job, not a correctness mechanism: Deduct and
// Resume enforce dates themselves regardless of status, so a missed or
// delayed sweep never risks the hard rules (see CLAUDE.md).
type Sweeper struct {
	repo *Repository
}

func NewSweeper(repo *Repository) *Sweeper {
	return &Sweeper{repo: repo}
}

// Run sweeps once immediately — so statuses aren't stale after downtime —
// then repeats on interval until ctx is done.
func (s *Sweeper) Run(ctx context.Context, interval time.Duration) {
	s.sweep(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweep(ctx)
		}
	}
}

func (s *Sweeper) sweep(ctx context.Context) {
	n, err := s.repo.CompletePastEndDate(ctx)
	if err != nil {
		log.Printf("sweep: marking past-end_date campaigns completed: %v", err)
		return
	}
	if n > 0 {
		log.Printf("sweep: marked %d campaign(s) completed", n)
	}
}
