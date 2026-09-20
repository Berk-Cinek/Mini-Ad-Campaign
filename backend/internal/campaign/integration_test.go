package campaign_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Berk-Cinek/Mini-Ad-Campaign/backend/internal/campaign"
)

// TestCampaignLifecycle drives one campaign through its full lifecycle over
// real HTTP: create, exhaust the budget via impressions (auto-pause),
// blocked resume, raise the budget, resume, end, blocked resume, stats,
// delete, and a 404 on the now-deleted campaign.
func TestCampaignLifecycle(t *testing.T) {
	ts := newTestServer(t)

	c := createCampaign(t, ts, 2)
	require.Equal(t, int64(2), c.Budget)
	require.Equal(t, int64(0), c.Spent)
	require.Equal(t, campaign.StatusActive, c.Status)

	resp := doRequest(t, ts, http.MethodPost, "/impression/"+idStr(c.ID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "impression 1")
	c = decode[campaign.Campaign](t, resp)
	require.Equal(t, int64(1), c.Spent)
	require.Equal(t, int64(1), c.Budget-c.Spent)
	require.Equal(t, campaign.StatusActive, c.Status)

	resp = doRequest(t, ts, http.MethodPost, "/impression/"+idStr(c.ID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "impression 2")
	c = decode[campaign.Campaign](t, resp)
	require.Equal(t, int64(2), c.Spent)
	require.Equal(t, int64(0), c.Budget-c.Spent)
	require.Equal(t, campaign.StatusPaused, c.Status, "should auto-pause once budget is exhausted")

	resp = doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/resume", nil)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "resume while spent >= budget")

	resp = doRequest(t, ts, http.MethodPatch, "/campaigns/"+idStr(c.ID), map[string]any{"budget": 5})
	require.Equal(t, http.StatusOK, resp.StatusCode, "raising budget")
	c = decode[campaign.Campaign](t, resp)
	require.Equal(t, int64(5), c.Budget)

	resp = doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/resume", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "resume after raising budget")
	c = decode[campaign.Campaign](t, resp)
	require.Equal(t, campaign.StatusActive, c.Status)

	resp = doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/end", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "end")
	c = decode[campaign.Campaign](t, resp)
	require.Equal(t, campaign.StatusCompleted, c.Status)

	resp = doRequest(t, ts, http.MethodPost, "/campaigns/"+idStr(c.ID)+"/resume", nil)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "resume after end")

	resp = doRequest(t, ts, http.MethodGet, "/stats/"+idStr(c.ID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, "stats")
	stats := decode[campaign.Stats](t, resp)
	require.Equal(t, int64(2), stats.Impressions)
	require.Equal(t, int64(2), stats.Spent)
	require.Equal(t, int64(3), stats.Remaining)
	require.Equal(t, int64(5), stats.Budget)
	require.Equal(t, campaign.StatusCompleted, stats.Status)

	resp = doRequest(t, ts, http.MethodDelete, "/campaigns/"+idStr(c.ID), nil)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "delete")

	resp = doRequest(t, ts, http.MethodGet, "/campaigns/"+idStr(c.ID), nil)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "get after delete")
}
