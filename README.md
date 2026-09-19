# Mini-Ad-Campaign

tests run against real Postgres because the invariants being tested are enforced in SQL, and mocking the repository would only test the stub.

## Known laxness: `budget` field type

`budget` is decoded as `json.Number`, which also accepts a quoted numeric string (e.g. `{"budget": "100"}`), not just a bare JSON number. This is intentional laxness, not a validation gap: a non-numeric value still fails (`400`), the frontend never sends budget as a string, and there's no auth/multi-client surface where a stricter type check would matter.