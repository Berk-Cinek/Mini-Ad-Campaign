Claude Code did the implementation. I ran it in plan mode for anything non-trivial: it proposed a plan, I reviewed and approved or rejected it, then it wrote the code and I read the diff before committing.

A Claude chat was used for one-off explanations: Go concepts I hadn't met before (why //go:embed can't reach a parent directory), and how TanStack Query's queries and mutations differ.
CLAUDE.md governed the implementation work. It's in the repo, and AI_CONTEXT.md explains how it was built and used.

How the loop worked

For every piece of work: decide what I wanted → plan mode → review the plan → implement → read the diff → verify by hand → commit → note anything that went wrong.

The plan-before-code step is what made this work. Most of the problems below were caught in a plan, before they became code I'd have had to unpick.

Where AI was wrong, and what I did
test suite that couldn't fail

I asked for a backend test suite: no mocks, real Postgres, each test inside a transaction that's always rolled back so the shared dev database stays clean. That part worked — 14 rows before the run, 14 after, despite dozens of creates across the suite.
But partway through writing it, it produced this:

go
func errorsAs(err error, target any) bool { return false }

and wired it into an assertion checking that the result was false. That test would have passed unconditionally while proving nothing. In the same file it wrote var _ = strings.TrimSpace purely to silence an unused-import error instead of removing the import.
It caught and cleaned up both itself before anything ran, and the final code compiled and passed. But it changed how I think about generated tests: "all tests pass" means nothing until I've read the tests. I went back through the suite looking for assertions that couldn't fail, and concluded the suite was too large to review properly — around 1,100 lines for what the case study lists as a bonus item.
So I reverted it. The revert is in the commit history.

A rule I never agreed to, shipped into the code
While planning PATCH, a plan stated as already-decided that campaign dates would only be editable while a campaign is paused. I hadn't decided that, and I said so explicitly and asked for it to be dropped.
It went into the code anyway, in both the service pre-check and the SQL WHERE clause. I only found it at the end, during a final read-only review pass over the whole backend, which flagged it as a CLAUDE.md rule the code didn't follow specifically against my own "do not decide open questions on your own" line. I kept the rule rather than removing it. dates affect whether impressions are accepted, so requiring a pause first is the safer behaviour, and pulling working backend logic out at the end of the project was the riskier choice.

AI catching a contradiction in my own rules

This one went the other way, and it is the best single piece of feedback I got.

My CLAUDE.md said a database constraint violation is always a bug: return 500 and log it. Reviewing the file, Claude Code pointed out that this prejudges the concurrency decision I had deliberately left open, because one legitimate mechanism is to let the CHECK (spent <= budget) constraint reject the overspend, and in that design the violation is expected contention rather than a bug, and should be 409.
It was right. I reworded the rule with an explicit exception tied to whichever mechanism I chose. When I later chose the conditional UPDATE, a constraint violation genuinely did become a bug, and the rule was correct as written.

Two bugs found by reading the code

Both looked completely correct at a glance.
start_date and end_date weren't actually required. A missing date decodes to Go's zero time, 0001-01-01, which passes both "end after start" and "end is in the future". You could create a campaign starting in year 1. Fixed with an IsZero() check.

The 200-character title limit used len(), which counts bytes, not characters. A Turkish title with several ş, ğ or ı would be wrongly rejected well short of 200 characters. Switched to utf8.RuneCountInString`

The race condition
I deliberately left the mechanism undecided in CLAUDE.md, with an instruction to propose options and wait, specifically so I would get a real proposal rather than a rubber stamp on a decision I had already made.
It proposed four options.

A, a single atomic conditional UPDATE. One statement does check and deduct together, with every precondition in the WHERE clause and the auto-pause in the same SET via `CASE WHEN spent + 1 >= budget`. Multi-instance correct, because Postgres locks the row and evaluates the whole WHERE clause against current committed state as one statement. Fewest round trips and shortest lock hold. The cost is that a failure is opaque, so it needs one follow-up SELECT to say why, paid only on failed requests.

B, `SELECT ... FOR UPDATE` inside an explicit transaction. Also correct across instances, and the failure reason is known immediately with no extra read. The cost is two round trips on every call, with the row lock held across both plus any Go-side work in between, roughly double the lock hold time under contention.

C, optimistic concurrency using compare-and-swap on `spent`. Correct, and no schema change, but under real contention on a single hot row most attempts lose and retry, which multiplies round trips. It suits low-contention fields; a hot campaign row is the opposite case.

D, an in-process `sync.Mutex` per campaign id. Listed explicitly to rule out. Each instance has its own lock table, so two instances can each pass their own local check and both write. It works in single-instance testing and breaks silently the moment there is more than one instance.

It also noted that a distributed lock, either a Postgres advisory lock or Redis, would work but only rebuilds with more moving parts the row-level serialization Postgres already provides, and Redis is out of scope in my CLAUDE.md anyway.
Was it good enough? Yes, and better than I expected on one specific point: it ruled out the mutex for the right reason, process-local state rather than vague robustness, and it named the real cost of option A rather than presenting it as free.

I chose A. The guarantee has to survive the process, because the state lives in Postgres and not in Go. It is also the same idiom already used for the budget-decrease guard and the resume guard, so there is one concurrency story in the codebase instead of three. The ambiguity cost is real but paid only on failures, and the follow-up SELECT is used purely to choose an error message, never to decide whether the write happens, so it does not reintroduce a race. The write has already atomically succeeded or failed. Database CHECK constraints stay a pure safety net rather than the mechanism, which is what keeps "a constraint violation is a bug, return 500" meaningful.

How much of the code did AI write
Most of it. I would estimate around 98% by line. Nearly all the Go and TypeScript was written by Claude Code from plans I approved.
What was mine: every design decision, including the data model, what "1 unit" means, the status transition rules, separate action endpoints instead of PATCH with a status field, a single fixed currency, and the concurrency mechanism. Rejecting plans, including a mock layer I didn't think earned its cost, an oversized test suite, the "VITE_API_BASE_URL" approach, and several proposed additions that were out of scope. Reviewing every diff, including the bugs above, which were caught by reading rather than by anything failing. And one full revert of AI-written code I couldn't vouch for.