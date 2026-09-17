
campaigns: create / list / get;
- when İmplemting Post /campaign i caught var campaigns []Campaign declares a nil slice. i thought it should have been [] as passing null to frontend is almost always wrong unless intentionally, when asked confirmed that; In Go, var campaigns []Campaign declares a nil slice. encoding/json marshals a nil slice as the JSON literal null, not []. So if List finds zero rows and  I declared the slice with var, GET /campaigns would return null instead of [] when there are no campaigns — which breaks any frontend code doing .map() on the response.

- also expended the scope slighty, adding GET /campaigns small edition so acceptable as it wont thouch other parts of the code but worth noticing.

- Bug found and fixed during verification: timestamps were coming back as +03:00 (host machine's local zone) instead of UTC — time.Time.MarshalJSON renders whatever Location pgx attaches to scanned values, and pgx doesn't default that to UTC. Fixed by normalizing all four timestamp fields to .UTC() inside Campaign.MarshalJSON, right where remaining is already computed. Confirmed fixed: "start_date":"2026-09-18T00:00:00Z".

- start_date / end_date weren't actually required. a missing date decodes to the zero time (0001-01-01), which passes both "end after start" and "end in the future". You could create a campaign starting in year 1. Fixed with an IsZero() check.

- the 200-character title limit used len(), which counts bytes, not characters. switched to utf8.RuneCountInString.

  