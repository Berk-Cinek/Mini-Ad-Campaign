# Mini-Ad-Campaign

## What it is

Mini-Ad-Campaign is a minimal platform for a retailer to manage ad campaigns and record impressions against a fixed budget. Each impression deducts one unit from the campaign's budget, and a campaign pauses itself the instant that budget is exhausted.

## Stack

Backend: Go 1.27, HTTP via the standard `net/http`, Postgres via `pgx`/`pgxpool`, schema migrations via goose (embedded into the binary and run on startup).

Frontend: React + TypeScript on Vite, server state via TanStack Query, plain CSS, client-side routing via `react-router`.

Infra: Docker Compose runs three services — Postgres, the Go backend (no published port; reachable only inside the compose network), and nginx, which serves the built frontend and proxies `/api/` to the backend.

## Running it

With Docker Compose, from a clean clone:

```
docker compose up
```

Open http://localhost:3000. nginx serves the frontend there and proxies `/api/*` through to the backend.


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

The impression endpoint accepts hundreds of concurrent requests against the same campaign, and the budget must never go negative. This is enforced with a single atomic conditional UPDATE (`Repository.Deduct` in `backend/internal/campaign/repository.go`): one SQL statement increments `spent` by 1 and, in the same `SET`, via `CASE WHEN spent + 1 >= budget THEN 'paused' ELSE status END`, flips the campaign to paused if that increment exhausts the budget. Every precondition — active, started, not ended, budget remaining — sits in the `WHERE` clause rather than in Go. Postgres evaluates that whole `WHERE` clause against the current committed row under lock as part of the one statement, so there is no window between checking and writing, regardless of how many separate backend processes are issuing the statement at once.

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

## Testing

The backend has two Go test files, both run against the real Postgres started by Docker Compose rather than a mock (`DATABASE_URL` unset skips them, so a bare `go test ./...` without Postgres running doesn't fail). `impression_test.go` fires 300 concurrent impressions at a single campaign and asserts the budget invariant end-to-end, in-process. `sweeper_test.go` asserts the background job flips a past-end-date campaign to completed and leaves a current one untouched. `loadtest/budget-loadtest.ps1` extends the same assertion over real HTTP and across multiple backend instances, as described above.

Coverage was kept small and directly readable rather than broad. An earlier, AI-generated backend test suite (roughly 1,100 lines) was reverted after a read turned up an assertion that could never fail — see `ai_sessions/AI_WORKFLOW.md`. The priority since has been proving the one invariant that's a hard rule — the budget never goes negative, under real concurrency, over real HTTP, across multiple instances — over maximizing line coverage. The frontend has no automated tests.

## Known limitations

Each backend instance runs its own database migrations on startup (`goose.Up` inside `main.go`'s `run()`) before it starts serving. With `docker compose up -d --scale backend=2`, or any multi-instance deployment, every instance attempts the same migration run at the same time. goose takes an advisory lock for exactly this case, so it should be safe, but I haven't verified the behaviour under simultaneous startup.

`budget` is decoded as `json.Number`, which accepts both a bare JSON number (`100`) and a quoted numeric string (`"100"`). This is intentional laxness rather than a validation gap.

The background sweep that marks past-end-date campaigns completed is a cosmetic catch-up job, not a correctness mechanism — impressions and resumes check dates themselves regardless of stored status. It runs on `SWEEP_INTERVAL` (default `1m`, in both local development and under `docker compose up`), so the stored `status` can lag up to a minute behind a campaign's actual end date. The frontend's `displayStatus()` (`frontend/src/lib/status.ts`) computes an `Ended` label from the dates directly, so the UI never shows a stale "Active"/"Paused" badge either way, but anything reading the API's raw `status` field can see it lag behind the true end date until the next sweep.

