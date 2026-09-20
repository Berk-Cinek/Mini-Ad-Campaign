# Mini-Ad-Campaign

## What it is

Mini-Ad-Campaign is a minimal platform for a retailer to manage ad campaigns and record impressions against a fixed budget. Each impression deducts one unit from the campaign's budget, and a campaign pauses itself the instant that budget is exhausted.

## Stack

- Backend
  - Go 1.27
  - HTTP via the standard `net/http`
  - Postgres access via `pgx`/`pgxpool`
- Frontend
  - React + TypeScript on Vite
  - Server state via TanStack Query
  - Plain CSS
  - Client-side routing via `react-router`
- Database
  - Postgres
  - Schema migrations via goose, embedded into the binary and run on startup
- Infra
  - Docker Compose runs three services: Postgres, the Go backend, and nginx
  - The backend has no published port; it is reachable only inside the compose network
  - nginx serves the built frontend and proxies `/api/` to the backend

## Running it

1. Clone the repo:

   ```
   git clone https://github.com/Berk-Cinek/Mini-Ad-Campaign
   cd Mini-Ad-Campaign
   ```

2. Start everything:

   ```
   docker compose up
   ```

3. Open http://localhost:3000. nginx serves the frontend there and proxies `/api/*` through to the backend.

## API

| Method | Path | Description | Success | Errors |
|---|---|---|---|---|
| POST | /campaigns | Create a campaign | 201 | 400 |
| GET | /campaigns | List campaigns, excluding soft-deleted, newest first | 200 | |
| GET | /campaigns/{id} | Get one campaign | 200 | 404 |
| PATCH | /campaigns/{id} | Update title, budget, and/or dates (partial update) | 200 | 400, 404, 409 |
| DELETE | /campaigns/{id} | Soft delete | 204 | 404 |
| POST | /campaigns/{id}/pause | active to paused | 200 | 404, 409 |
| POST | /campaigns/{id}/resume | paused to active | 200 | 404, 409 |
| POST | /campaigns/{id}/end | active or paused to completed | 200 | 404, 409 |
| POST | /impression/{id} | Record one impression; deducts one budget unit | 200 | 404, 409 |
| GET | /stats/{id} | impressions, spent, remaining, budget, status | 200 | 404 |

Every endpoint above also returns 400 if `{id}` isn't a valid integer, and 500 with `{"error": "internal server error"}` on an unexpected failure.

## Race condition

The impression endpoint accepts hundreds of concurrent requests against the same campaign, and the budget must never go negative. This is enforced by a single atomic conditional UPDATE whose `WHERE` clause holds every precondition, so Postgres's row lock makes check-and-write one indivisible step, even across multiple backend processes.

The detail: the statement is `Repository.Deduct` in `backend/internal/campaign/repository.go`. One SQL statement increments `spent` by 1 and, in the same `SET`, via `CASE WHEN spent + 1 >= budget THEN 'paused' ELSE status END`, flips the campaign to paused if that increment exhausts the budget. Every precondition — active, started, not ended, budget remaining — sits in the `WHERE` clause rather than in Go. Postgres evaluates that whole `WHERE` clause against the current committed row under lock as part of the one statement, so there is no window between checking and writing, regardless of how many separate backend processes are issuing the statement at once.

If zero rows match, nothing is written, and the service issues one follow-up `SELECT` purely to work out which precondition failed and choose a 409 message — that read never decides whether the write happens, so it cannot reintroduce a race.

The same idiom guards the budget-decrease case in `Update` and the resume guard in `Resume`, so there is one concurrency mechanism in the codebase, not three.

An in-process mutex was considered and rejected. A `sync.Mutex`, or any per-process lock, only serializes goroutines inside one Go process's memory; it cannot see what another process is doing. Each additional backend instance would carry its own independent lock table, so two instances could each pass their own local check and both write — correct under single-instance testing, and silently broken the moment there is more than one instance, which is exactly the failure Postgres's row-level locking rules out without any extra machinery on the application side.

This is proven at three levels. `backend/internal/campaign/impression_test.go` (`TestConcurrentImpressions_NeverOverspend`) fires 300 concurrent goroutines at a budget-50 campaign inside a single process and asserts exactly 50 succeed, `spent == budget`, and the campaign ends up paused. `loadtest/budget-loadtest.ps1` repeats the same assertion over real HTTP: it creates a budget-100 campaign, fires 2000 concurrent `POST /impression/{id}` requests at it with `hey`, and checks `GET /stats/{id}` for `spent == 100`, `remaining == 0`, `status == paused`. Finally, that same load test is re-run against `docker compose up -d --scale backend=2` through nginx — two separate backend processes sharing one Postgres instance — which is the scenario an in-process mutex cannot survive but the conditional UPDATE can.

Output of the scaled (`--scale backend=2`) load test run:

```
Details (average, fastest, slowest):
  DNS+dialup:   0.0011 secs, 0.0000 secs, 0.0165 secs
  DNS-lookup:   0.0009 secs, 0.0000 secs, 0.0148 secs
  req write:    0.0001 secs, 0.0000 secs, 0.0066 secs
  resp wait:    0.0165 secs, 0.0007 secs, 0.1587 secs
  resp read:    0.0001 secs, 0.0000 secs, 0.0020 secs

Status code distribution:
  [200] 100 responses
  [409] 1900 responses


hey status code distribution: 200=100 (expected 100), 409=1900 (expected 1900), 500=0 (expected 0)

Fetching GET /stats/3 ...

spent:     actual=100     expected=100      OK
remaining: actual=0  expected=0        OK
status:    actual=paused  expected=paused   OK

PASS
```

### Running the tests

    cd backend
    $env:DATABASE_URL = "postgresql://campaign:campaign@localhost:5432/campaign?sslmode=disable"
    go test ./... -count=1

## Testing

The backend tests run against the real Postgres started by Docker Compose rather than mocks. If `DATABASE_URL` is unset they are skipped, so a bare `go test ./...` without Postgres running doesn't fail. See "Running the tests" above for the command.

- `impression_test.go` fires 300 concurrent impressions at one campaign in-process and asserts the budget invariant.
- `integration_test.go` drives one campaign through its full lifecycle over real HTTP: create, exhaust the budget, auto-pause, blocked resume, budget raise, resume, end, stats, delete.
- `service_test.go` covers the conflict paths that need seeded state: resume with no budget left or a passed end date, a budget decrease below spent, and impressions on deleted, paused or not-yet-started campaigns.
- `sweeper_test.go` covers the background completion job.
- `loadtest/budget-loadtest.ps1` extends the budget assertion over HTTP and across two backend instances.

Tests using HTTP run inside a transaction that is always rolled back, so they leave no rows behind; the campaign row count is unchanged after a full run.

## Known limitations

- **Migrations run on every instance at startup, and simultaneous startup is unverified.** Each backend instance runs `goose.Up` inside `main.go`'s `run()` before it starts serving, so with `docker compose up -d --scale backend=2`, or any multi-instance deployment, every instance attempts the same migration run at the same time. goose takes an advisory lock for exactly this case, so it should be safe, but I haven't verified the behaviour under simultaneous startup.
- **`budget` accepts a bare number or a quoted numeric string.** It is decoded as `json.Number`, so both `100` and `"100"` are valid. This is intentional laxness rather than a validation gap.
- **The stored `status` can lag up to a minute behind a campaign's end date.** The background sweep that marks past-end-date campaigns completed is a cosmetic catch-up job, not a correctness mechanism — impressions and resumes check dates themselves regardless of stored status. It runs on `SWEEP_INTERVAL` (default `1m`, in both local development and under `docker compose up`). The frontend's `displayStatus()` (`frontend/src/lib/status.ts`) computes an `Ended` label from the dates directly, so the UI never shows a stale "Active"/"Paused" badge either way, but anything reading the API's raw `status` field can see it lag behind the true end date until the next sweep.
