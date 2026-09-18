
campaigns: create / list / get;
- when İmplemting Post /campaign i caught var campaigns []Campaign declares a nil slice. i thought it should have been [] as passing null to frontend is almost always wrong unless intentionally, when asked confirmed that; In Go, var campaigns []Campaign declares a nil slice. encoding/json marshals a nil slice as the JSON literal null, not []. So if List finds zero rows and  I declared the slice with var, GET /campaigns would return null instead of [] when there are no campaigns — which breaks any frontend code doing .map() on the response.

- also expended the scope slighty, adding GET /campaigns small edition so acceptable as it wont thouch other parts of the code but worth noticing.

- Bug found and fixed during verification: timestamps were coming back as +03:00 (host machine's local zone) instead of UTC — time.Time.MarshalJSON renders whatever Location pgx attaches to scanned values, and pgx doesn't default that to UTC. Fixed by normalizing all four timestamp fields to .UTC() inside Campaign.MarshalJSON, right where remaining is already computed. Confirmed fixed: "start_date":"2026-09-18T00:00:00Z".

- start_date / end_date weren't actually required. a missing date decodes to the zero time (0001-01-01), which passes both "end after start" and "end in the future". You could create a campaign starting in year 1. Fixed with an IsZero() check.

- the 200-character title limit used len(), which counts bytes, not characters. switched to utf8.RuneCountInString.

The awnser to the race condtion solution;
Option A — Single atomic conditional UPDATE

  One SQL statement does the check-and-deduct together:
  UPDATE campaigns
  SET
      spent  = spent + 1,
      status = CASE WHEN spent + 1 >= budget THEN 'paused' ELSE status END,
      updated_at = now()
  WHERE id = $1
      AND deleted_at IS NULL
      AND status = 'active'
      AND start_date <= now()
      AND end_date > now()
      AND spent < budget
  RETURNING spent, budget, status, ...
  Pros: Multi-instance correct (Postgres locks the row and evaluates the whole WHERE clause against current committed state as one statement — instances are
  serialized by Postgres, not app coordination). Fewest round trips on the success path. Shortest lock hold time per request → best throughput under
  contention. Same idiom already used for PATCH's budget-decrease guard.
  Cons: On failure, the reason is opaque (missing/deleted vs. not active vs. bad dates vs. exhausted) — needs one follow-up SELECT to disambiguate, paid
  only on failed requests.

  Option B — SELECT ... FOR UPDATE inside an explicit transaction

  Lock the row, check conditions in Go, then update, all in one transaction.
  Pros: Multi-instance correct (FOR UPDATE blocks a second transaction's lock attempt on the same row until the first commits). Failure reason is known
  immediately in Go — no extra disambiguating read.
  Cons: Two round trips on every call, not just failures. Row lock held across both round trips plus any Go-side computation in between — roughly double the
  lock-hold time of Option A under contention. More code (explicit transaction lifecycle, held connection).

  Option C — Optimistic concurrency (compare-and-swap on spent)

  Read spent, compute new value, UPDATE ... WHERE spent=$old, retry on conflict. No new column needed — reuses spent itself as the CAS token.
  Pros: Multi-instance correct (each CAS UPDATE is atomic regardless of instance). No schema change.
  Cons: Under real contention on one row, many attempts lose the race and retry — multiplies round trips well beyond A or B. This pattern suits
  low-contention fields; a hot campaign row under load is the opposite case. Needs a max-retry/backoff policy to avoid spinning.

  Option D — In-process lock (sync.Mutex per campaign id)

  Pros: Simple, fast within a single process.
  Cons: Not multi-instance correct — each backend instance has its own independent lock table; two instances can each pass their own local check and both
  write, overspending. Works fine in single-instance testing and silently breaks the moment there's more than one instance. Listed here specifically to rule
  it out.

  Not listed as real options: a distributed lock (Postgres advisory lock or Redis) would work but only reconstructs, with more moving parts, the same
  row-level serialization A/B/C already get from Postgres for free — and a Redis-based version is a new dependency requiring separate approval anyway. A
  message-queue-based serializer is similarly disproportionate for "hundreds of concurrent requests" on one row.

  - Choose Option A for multiple reasones, since go does not hold the actual state the db does the guarentee has to survive the process to be effective, also simplicity is key works wonderfully only double returning on failures to resolve ambiguity.

  Test Generation:

  - Generated a comprehensive testing suit for all endpoints + an integration test suite without mocks: real Postgres, each test inside a transaction that's always rolled back, so the shared dev DB stays clean. Worked, 14 rows before, 14 after, despite dozens of creates across the suite. The one change was impression_test.go's own soft-deleted row, which is expected since that test can't run in a wrapping transaction (it needs genuinely concurrent committing transactions to prove the row locking). The thing I need to remember from this session: at one point it wrote placeholder functions that always returned false —

        func errorsAs(err error, target any) bool { return false }

    and wired one into an assertion checking for false. That test would have passedunconditionally while proving nothing. It also wrote `var _ = strings.TrimSpace` purely to silence an unused-import error instead of removing the import. It caught and cleaned up both itself before anything ran, and the final code is correct, but it's the clearest example yet that "all tests pass" means nothing until I've read the tests. Went back through service_test.go and handlers_test.go looking for
    assertions that can't fail.

    Frontend Scafolding:

    - tried to use vite_api_base_url instead of a proxy as defined int he CLAUDE.md file, since it was ngix my guess is it thought the proxy was for prod only and defaulted to base url approach