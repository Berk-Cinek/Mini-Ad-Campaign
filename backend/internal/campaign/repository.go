package campaign

import (
	"context"

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
