package campaign_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestServiceConflicts exercises the conflict (409) and not-found (404)
// paths that require specific pre-seeded state to reach.
func TestServiceConflicts(t *testing.T) {
	t.Run("resume when spent >= budget", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)
		seedCampaign(t, ts, c.ID, "paused", 10, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))

		resp := doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/resume", nil)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("resume when end_date has passed", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)
		seedCampaign(t, ts, c.ID, "paused", 0, time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))

		resp := doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/resume", nil)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("PATCH budget below spent", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)
		seedCampaign(t, ts, c.ID, "active", 5, c.StartDate, c.EndDate)

		resp := doRequest(t, ts, http.MethodPatch, "/campaigns/"+idStr(c.ID), map[string]any{"budget": 3})
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("impression on a soft-deleted campaign", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)

		resp := doRequest(t, ts, http.MethodDelete, "/campaigns/"+idStr(c.ID), nil)
		require.Equal(t, http.StatusNoContent, resp.StatusCode, "setup delete")

		resp = doRequest(t, ts, http.MethodPost, "/impression/"+idStr(c.ID), nil)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("impression on a paused campaign", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)
		seedCampaign(t, ts, c.ID, "paused", 0, c.StartDate, c.EndDate)

		resp := doRequest(t, ts, http.MethodPost, "/impression/"+idStr(c.ID), nil)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("impression before start_date", func(t *testing.T) {
		ts := newTestServer(t)
		c := createCampaign(t, ts, 10)
		seedCampaign(t, ts, c.ID, "active", 0, time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))

		resp := doRequest(t, ts, http.MethodPost, "/impression/"+idStr(c.ID), nil)
		require.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}
