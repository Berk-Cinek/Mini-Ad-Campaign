-- +goose Up
CREATE TABLE campaigns (
    id          BIGSERIAL PRIMARY KEY,
    title       TEXT NOT NULL,
    budget      BIGINT NOT NULL,
    spent       BIGINT NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'active',
    start_date  TIMESTAMPTZ NOT NULL,
    end_date    TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,

    CONSTRAINT campaigns_budget_positive CHECK (budget > 0),
    CONSTRAINT campaigns_spent_nonnegative CHECK (spent >= 0),
    CONSTRAINT campaigns_spent_le_budget CHECK (spent <= budget),
    CONSTRAINT campaigns_end_after_start CHECK (end_date > start_date),
    CONSTRAINT campaigns_status_valid CHECK (status IN ('active', 'paused', 'completed'))
);

-- +goose Down
DROP TABLE campaigns;
