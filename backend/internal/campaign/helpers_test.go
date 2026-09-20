package campaign_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/Berk-Cinek/Mini-Ad-Campaign/backend/internal/campaign"
)

type testServer struct {
	url string
	tx  pgx.Tx
}

// newTestServer opens a pool, begins a transaction that always rolls back,
// and serves the campaign API on top of it. Cleanups run LIFO, so the server
// closes first, then the transaction rolls back, then the pool closes.
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; run against the Postgres started by docker compose")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err, "connecting to database")
	t.Cleanup(pool.Close)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "beginning transaction")
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })

	repo := campaign.NewRepository(tx)
	svc := campaign.NewService(repo)
	h := campaign.NewHandler(svc)

	mux := http.NewServeMux()
	campaign.RegisterRoutes(mux, h)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return &testServer{url: srv.URL, tx: tx}
}

// createCampaign posts a new campaign with the given budget and a window
// that has already started and ends an hour from now.
func createCampaign(t *testing.T, ts *testServer, budget int64) campaign.Campaign {
	t.Helper()
	body := map[string]any{
		"title":      "test campaign",
		"budget":     budget,
		"start_date": time.Now().Add(-time.Hour),
		"end_date":   time.Now().Add(time.Hour),
	}
	resp := doRequest(t, ts, http.MethodPost, "/campaigns", body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "creating campaign")
	return decode[campaign.Campaign](t, resp)
}

// seedCampaign overwrites status/spent/dates directly via SQL on the test's
// transaction, bypassing the service layer to set up states the API can't
// reach on its own (e.g. a paused campaign with spent already at budget).
func seedCampaign(t *testing.T, ts *testServer, id int64, status string, spent int64, start, end time.Time) {
	t.Helper()
	_, err := ts.tx.Exec(context.Background(),
		`UPDATE campaigns SET status = $1, spent = $2, start_date = $3, end_date = $4 WHERE id = $5`,
		status, spent, start, end, id)
	require.NoError(t, err, "seeding campaign %d", id)
}

func doRequest(t *testing.T, ts *testServer, method, path string, body any) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err, "marshaling request body")
		reqBody = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, ts.url+path, reqBody)
	require.NoError(t, err, "building request")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "performing request")
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var out T
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out), "decoding response body")
	return out
}

func idStr(id int64) string {
	return strconv.FormatInt(id, 10)
}
