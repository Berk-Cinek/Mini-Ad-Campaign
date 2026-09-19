# Mini Ad Campaign Platform

## project

Minimal platform for a retailer to manage ad campaigns and record impressions.
Each impression deducts budget; stats show spend and remaining budget.
simplicity is the goal adding features needs to be justified to me.
small but solid is better then big but messy.

## Out of scope, do not add
- Auth, users, payments, analytics beyond /stats
- Any new dependency without asking me first

## Tech Stack
- Backend: Go 1.27, migrations to be managed with goose, PostgresSQL for DB, API with 'net/http', DB access with pgx + pgxpool
- Backend tests: go test against the Postgres started by Docker Compose
- Frontend: React + TypeScript (Vite), TanStack querry for server stat, styling done in plain css 
- Infra: Docker compose:postgres backend, nginx frontend

## Hard Rules
These must never be broken. If a task seems to require breaking one, stop and ask.
- The budget must never go negative. This requirement is fixed. *
- The impression endpoint receives hundreds of concurrent requests. When the last unit of budget is used, the campaign pauses. Mechanism: a single atomic conditional UPDATE (deducts and pauses in one statement, gated by a WHERE clause checking status/dates/remaining budget) — race-safe across multiple backend instances because Postgres evaluates the WHERE clause against the current row under lock as part of that one statement, not the application.
- Decreasing a campaign's budget (via PATCH) uses the same mechanism: a WHERE-clause guard (`spent <= new budget`) on the UPDATE, for the same reason.
- Resuming a campaign uses the same mechanism: a WHERE-clause guard (`spent < budget AND end_date > now()`) on the UPDATE, for the same reason.
- Money is an integer: BIGINT in the database, int64 in Go, integer in JSON. Never float.
- Database constraints are the safety net: budget > 0, spent >= 0, spent <= budget, end_date > start_date, status limited to the three values.
- soft delete only. Deleting sets deleted_at. Every query filters deleted_at IS NULL. No DELETE statements.
Remaining budget is computed (budget - spent), never stored.
- every UPDATE sets updated_at = now(). Postgres does not do this automatically.

## Domain

campaign:
- id: BIGSERIAL
- title: text, required
- budget: BIGINT, whole TRY
- spent: BIGINT, starts at 0
- status: active | paused | completed
- start_date, end_date: TIMESTAMPTZ
- created_at, updated_at: TIMESTAMPTZ
- deleted_at: TIMESTAMPTZ, null unless soft-deleted

## Domain Decisions

- currency whole units. ('BIGINT' in DB, 'int64' in Go). One impression costs exactly 1 unit, currency is always Turkish lira (TRY). It is fixed and not stored.
- only active campaigns accept impressions.
- impressions are derived from spent, not stored as a separate counter (1 impression = 1 unit spent, always).
- dates: stored as timestamptz in utc end_Date must be after start_date and in future creation impressions are accepted only when start_date is <= now < end_date (enforced on patch)
- budget increases are allowed anytime on active and paused campaigns.
- budget decreases are allowed only if the new budget is still >= spent.
- completed campaigns cannot be edited in any way. completion is terminal and cannot be reversed.
- increasing the budget never resumes a paused campaign automatically.
- start_date/end_date can only be changed via PATCH while the campaign is paused; sending either while active or completed returns 409. (Resolved open question.)
- ending a campaign early is allowed. There is no permission check because there is no auth.
- a background job runs every minute and marks campaigns past end_date as completed. Correctness never depends on this job, because the impression logic checks dates itself.
- soft-deleted campaigns return 404 on every endpoint, including a second DELETE. No restore feature.
- On PATCH id may not be thouched aswell as status, status is changed through dedicated endpoints.

## API

GET - /campaigns - list, exclude soft-deletes, newest first(created_at)
GET - /campaigns/{id} - find one campaing by id 404 if missing or deleted
POST - /campaigns - create, validates (rules on line 77) fields
PATCH - /campaigns/{id} - update title, budget, dates only. Sending status returns 400
POST - /campaigns/{id}/pause - active => paused. 404 missing/deleted, 409 if not active
POST - /campaigns/{id}/resume - paused => active. 404 missing/deleted, 409 if not paused, no budget left, or end date passed
POST - /campaigns/{id}/end - active or paused - completed. 404 missing/deleted, 409 if already completed
DELETE - /campaigns/{id} - soft delete by id return 204
POST - /impression/{id} - 200 ok, 404 missing/deleted, 409 not deductible. each call deducts 1 unit from campaign budget
GET - /stats/{id} - impressions, spent, remaining, budget, status

PATCH is a partial update: only the fields sent are changed.
Cross-field checks use the stored value for fields not sent. Example: a new end_date is compared with the existing start_date.
Dates in JSON are ISO 8601 / RFC 3339 strings in UTC.
Status codes: 400 invalid input, 404 missing or deleted, 409 not allowed in the current state, 500 unexpected error.
Errors are JSON: '{"error": "readable message"}' with a correct status code.

Validation (400 on failure, checked before any database write):
- title: required, not empty after trimming, max 200 characters
- budget: required, whole number >= 1 (decimals rejected)
- start_date, end_date: required, RFC 3339 format, end_date after start_date, end_date in the future
- clients cannot set status, spent, id or timestamps; unknown JSON fields return 400
- PATCH applies the same rules to the fields it receives
- A database constraint violation is unexpected: return 500 and log it. The chosen concurrency mechanism (WHERE-clause guards on conditional UPDATEs) never relies on catching a constraint violation, so constraints stay a pure safety net — any violation is a bug, not an expected 409 path.

## Open questions

Do not decide these on your own. Ask me when they become relevant.

(none currently)

## Code style
- TypeScript: no 'any'. Types in 'types.ts' mirror the JSON responses exactly.
- Server state only through TanStack Query.

## How to work with me
- For anything non-trivial, propose a short plan first and wait for my OK.
- Before changing the impression query, schema, or status transitions,
  explain why the change is race-safe.
- Don't silently work around these rules. If you think one is wrong, say so and argue.
- Keep diffs small and focused; don't refactor unrelated code.
- If a requirement is ambiguous, ask instead of inventing scope.
- After a change, tell me the exact command to verify it.

## Definition of done
- `docker compose up` works from a clean clone
- README updated if a decision changed
- a Go test proves the budget never goes negative under concurrent impressions
