package campaign

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const campaignColumns = "id, title, budget, spent, status, start_date, end_date, created_at, updated_at"

func (r *Repository) Create(ctx context.Context, c Campaign) (Campaign, error) {
	query := `
		INSERT INTO campaigns (title, budget, start_date, end_date)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + campaignColumns

	var out Campaign
	err := r.pool.QueryRow(ctx, query, c.Title, c.Budget, c.StartDate, c.EndDate).Scan(
		&out.ID, &out.Title, &out.Budget, &out.Spent, &out.Status,
		&out.StartDate, &out.EndDate, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return Campaign{}, err
	}
	return out, nil
}

func (r *Repository) List(ctx context.Context) ([]Campaign, error) {
	query := `
		SELECT ` + campaignColumns + `
		FROM campaigns
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := []Campaign{} // non-nil so a zero-row result marshals to [] not null
	for rows.Next() {
		var c Campaign
		if err := rows.Scan(
			&c.ID, &c.Title, &c.Budget, &c.Spent, &c.Status,
			&c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return campaigns, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (Campaign, error) {
	query := `
		SELECT ` + campaignColumns + `
		FROM campaigns
		WHERE id = $1 AND deleted_at IS NULL`

	var c Campaign
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Title, &c.Budget, &c.Spent, &c.Status,
		&c.StartDate, &c.EndDate, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return Campaign{}, err
	}
	return c, nil
}

// Update applies a partial update, using COALESCE so an absent (nil) field
// keeps the column's current value. Both invariants below are enforced as
// WHERE conditions rather than checked beforehand in Go, so they're evaluated
// by Postgres atomically against the current row under lock as part of this
// single statement — race-safe across multiple backend instances without any
// extra locking on our side:
//   - spent <= COALESCE(budget, budget): a no-op when budget isn't being
//     changed (spent <= budget already holds via the CHECK constraint), and
//     the real budget-decrease-vs-spent guard when it is.
//   - dates only touchable while paused: vacuously true when neither date
//     param is set, otherwise requires status = 'paused'.
//
// Zero matching rows surfaces as pgx.ErrNoRows via Scan, same as GetByID.
func (r *Repository) Update(ctx context.Context, id int64, title *string, budget *int64, startDate, endDate *time.Time) (Campaign, error) {
	// Params are explicitly cast (::text/::bigint/::timestamptz) because when
	// every optional field but one is nil, Postgres can't always infer a
	// placeholder's type from COALESCE/IS NULL context alone and rejects the
	// query with "could not determine data type of parameter" (42P08).
	query := `
		UPDATE campaigns
		SET
			title      = COALESCE($2::text, title),
			budget     = COALESCE($3::bigint, budget),
			start_date = COALESCE($4::timestamptz, start_date),
			end_date   = COALESCE($5::timestamptz, end_date),
			updated_at = now()
		WHERE id = $1
			AND deleted_at IS NULL
			AND status <> 'completed'
			AND spent <= COALESCE($3::bigint, budget)
			AND (($4::timestamptz IS NULL AND $5::timestamptz IS NULL) OR status = 'paused')
		RETURNING ` + campaignColumns

	var out Campaign
	err := r.pool.QueryRow(ctx, query, id, title, budget, startDate, endDate).Scan(
		&out.ID, &out.Title, &out.Budget, &out.Spent, &out.Status,
		&out.StartDate, &out.EndDate, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return Campaign{}, err
	}
	return out, nil
}

// SoftDelete never issues a DELETE statement — it sets deleted_at, and every
// query elsewhere filters deleted_at IS NULL. No status restriction: any
// non-deleted campaign, regardless of status, can be soft-deleted.
func (r *Repository) SoftDelete(ctx context.Context, id int64) error {
	query := `
		UPDATE campaigns
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id`

	var returnedID int64
	return r.pool.QueryRow(ctx, query, id).Scan(&returnedID)
}
