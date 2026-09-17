package campaign

import (
	"encoding/json"
	"time"
)

type Campaign struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Budget    int64     `json:"budget"`
	Spent     int64     `json:"spent"`
	Status    string    `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MarshalJSON adds a computed "remaining" field. Remaining budget is never
// stored (budget - spent), so it's derived here rather than on the struct.
func (c Campaign) MarshalJSON() ([]byte, error) {
	type Alias Campaign // defined type, not an alias — drops MarshalJSON so this doesn't recurse

	// pgx returns scanned timestamps in the driver/host's local Location; the
	// instant is correct but json's RFC3339 output would print that offset
	// instead of "Z". Normalize to UTC here so every response is UTC.
	c.StartDate = c.StartDate.UTC()
	c.EndDate = c.EndDate.UTC()
	c.CreatedAt = c.CreatedAt.UTC()
	c.UpdatedAt = c.UpdatedAt.UTC()

	return json.Marshal(struct {
		Alias
		Remaining int64 `json:"remaining"`
	}{
		Alias:     Alias(c),
		Remaining: c.Budget - c.Spent,
	})
}

type CreateInput struct {
	Title     string      `json:"title"`
	Budget    json.Number `json:"budget"`
	StartDate time.Time   `json:"start_date"`
	EndDate   time.Time   `json:"end_date"`
}
