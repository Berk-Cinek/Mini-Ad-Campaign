import { useState } from 'react';
import type { FormEvent } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';

import { request } from '../api/client';
import StatusBadge from '../components/StatusBadge';
import { toDatetimeLocalValue } from '../lib/datetime';
import type { Campaign, Stats, UpdateCampaignInput } from '../types';
import './CampaignDetailPage.css';

export default function CampaignDetailPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const campaignQuery = useQuery({
    queryKey: ['campaign', id],
    queryFn: () => request<Campaign>(`/api/campaigns/${id}`),
  });

  const statsQuery = useQuery({
    queryKey: ['stats', id],
    queryFn: () => request<Stats>(`/api/stats/${id}`),
    refetchInterval: 2000,
  });

  // Tracks which fetched campaign the edit form's fields currently reflect,
  // so they re-sync whenever the query returns a genuinely new object (first
  // load, a save, or another action's refetch) without needing an effect —
  // setting state during render like this is React's documented pattern for
  // "adjust state when a prop/query result changes".
  const [syncedCampaign, setSyncedCampaign] = useState<Campaign | null>(null);
  const [budgetDelta, setBudgetDelta] = useState('0');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  if (campaignQuery.data && campaignQuery.data !== syncedCampaign) {
    setSyncedCampaign(campaignQuery.data);
    setBudgetDelta('0');
    setStartDate(toDatetimeLocalValue(new Date(campaignQuery.data.start_date)));
    setEndDate(toDatetimeLocalValue(new Date(campaignQuery.data.end_date)));
  }

  function invalidateDetail() {
    queryClient.invalidateQueries({ queryKey: ['campaign', id] });
    queryClient.invalidateQueries({ queryKey: ['stats', id] });
  }

  const edit = useMutation({
    mutationFn: (input: UpdateCampaignInput) =>
      request<Campaign>(`/api/campaigns/${id}`, { method: 'PATCH', body: input }),
    onSuccess: invalidateDetail,
  });

  const pause = useMutation({
    mutationFn: () => request<Campaign>(`/api/campaigns/${id}/pause`, { method: 'POST' }),
    onSuccess: invalidateDetail,
  });

  const resume = useMutation({
    mutationFn: () => request<Campaign>(`/api/campaigns/${id}/resume`, { method: 'POST' }),
    onSuccess: invalidateDetail,
  });

  const end = useMutation({
    mutationFn: () => request<Campaign>(`/api/campaigns/${id}/end`, { method: 'POST' }),
    onSuccess: invalidateDetail,
  });

  const del = useMutation({
    mutationFn: () => request<void>(`/api/campaigns/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['campaigns'] });
      navigate('/');
    },
  });

  const impression = useMutation({
    mutationFn: () => request<Campaign>(`/api/impression/${id}`, { method: 'POST' }),
    onSuccess: invalidateDetail,
  });

  if (!id) {
    return <p>Invalid campaign id.</p>;
  }

  if (campaignQuery.isLoading) {
    return <p>Loading…</p>;
  }

  if (campaignQuery.error) {
    return <p>{campaignQuery.error.message}</p>;
  }

  const campaign = campaignQuery.data;
  if (!campaign) {
    return <p>Campaign not found.</p>;
  }

  const isCompleted = campaign.status === 'completed';

  // Both derived from the same hypothetical, unsaved state — shown together
  // so they can't be read as contradicting the still-live "Remaining" figure
  // in the Live Stats panel below, which reflects the actual, saved budget
  // until this form is actually submitted.
  const previewBudget = campaign.budget + (Number(budgetDelta) || 0);
  const previewRemaining = previewBudget - campaign.spent;

  function handleSave(e: FormEvent) {
    e.preventDefault();
    if (!campaign || !startDate || !endDate) {
      return;
    }

    // Only send fields that actually changed. The backend treats a date
    // field's mere presence in the request as "touching" it (gating on
    // paused-only), so resending the current dates unchanged alongside a
    // budget-only edit would incorrectly trip that guard on an active
    // campaign — this keeps PATCH's partial-update contract honest.
    const input: UpdateCampaignInput = {};
    const delta = Number(budgetDelta) || 0;
    if (delta !== 0) {
      input.budget = campaign.budget + delta;
    }
    if (startDate !== toDatetimeLocalValue(new Date(campaign.start_date))) {
      input.start_date = new Date(startDate).toISOString();
    }
    if (endDate !== toDatetimeLocalValue(new Date(campaign.end_date))) {
      input.end_date = new Date(endDate).toISOString();
    }

    if (Object.keys(input).length === 0) {
      return;
    }

    edit.mutate(input);
  }

  return (
    <div>
      <h1>{campaign.title}</h1>

      <div className="detail-card">
        <div className="detail-grid">
          <div className="detail-info">
            <p>
              Status: <StatusBadge campaign={campaign} />
            </p>

            <form className="edit-form" onSubmit={handleSave}>
              <label>
                Adjust budget by
                <input
                  type="number"
                  value={budgetDelta}
                  onChange={(e) => setBudgetDelta(e.target.value)}
                  step={1}
                  disabled={isCompleted}
                />
              </label>
              {previewBudget === campaign.budget ? (
                <p className="field-hint">Positive to add, negative to reduce.</p>
              ) : (
                <p className="field-hint">
                  If saved: budget {previewBudget.toLocaleString()}, remaining {previewRemaining.toLocaleString()}.
                </p>
              )}

              <label>
                Start date
                <input
                  type="datetime-local"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  required
                  disabled={isCompleted}
                />
              </label>

              <label>
                End date
                <input
                  type="datetime-local"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  required
                  disabled={isCompleted}
                />
              </label>

              {edit.error && <p className="action-error">{edit.error.message}</p>}

              <button className="btn-primary" type="submit" disabled={isCompleted || edit.isPending}>
                {edit.isPending ? 'Saving…' : 'Save changes'}
              </button>
            </form>
          </div>

          <div className="detail-side">
            <section className="detail-stats">
              <h2>Live Stats</h2>
              {statsQuery.isLoading && <p>Loading…</p>}
              {statsQuery.error && <p>{statsQuery.error.message}</p>}
              {statsQuery.data && (
                <>
                  <p>Impressions: {statsQuery.data.impressions.toLocaleString()}</p>
                  <p>Spent: {statsQuery.data.spent.toLocaleString()}</p>
                  <p>Remaining: {statsQuery.data.remaining.toLocaleString()}</p>
                </>
              )}
            </section>

            <section className="detail-actions">
              <h2>Actions</h2>

              <div className="actions-row">
                <div className="action-group">
                  <button
                    className="btn-primary"
                    type="button"
                    title="Stops the campaign from serving impressions until resumed."
                    disabled={isCompleted || pause.isPending}
                    onClick={() => pause.mutate()}
                  >
                    {pause.isPending ? 'Pausing…' : 'Pause'}
                  </button>
                  {pause.error && <p className="action-error">{pause.error.message}</p>}
                </div>

                <div className="action-group">
                  <button
                    className="btn-primary"
                    type="button"
                    title="Reactivates a paused campaign, if budget and dates allow."
                    disabled={isCompleted || resume.isPending}
                    onClick={() => resume.mutate()}
                  >
                    {resume.isPending ? 'Resuming…' : 'Resume'}
                  </button>
                  {resume.error && <p className="action-error">{resume.error.message}</p>}
                </div>

                <div className="action-group">
                  <button
                    className="btn-danger"
                    type="button"
                    title="Ends the campaign now, permanently. This can't be undone."
                    disabled={isCompleted || end.isPending}
                    onClick={() => end.mutate()}
                  >
                    {end.isPending ? 'Ending…' : 'End'}
                  </button>
                  {end.error && <p className="action-error">{end.error.message}</p>}
                </div>

                <div className="action-group">
                  <button
                    className="btn-danger"
                    type="button"
                    title="Soft-deletes the campaign; it disappears from every view."
                    disabled={del.isPending}
                    onClick={() => del.mutate()}
                  >
                    {del.isPending ? 'Deleting…' : 'Delete'}
                  </button>
                  {del.error && <p className="action-error">{del.error.message}</p>}
                </div>
              </div>
            </section>
          </div>
        </div>

        <div className="debug-section">
          <h2>Debug / Simulation</h2>
          <p>
            In production, impressions are recorded by the ad server, not by hand. This
            demonstrates budget deduction and auto-pause.
          </p>
          <button
            className="btn-primary"
            type="button"
            disabled={impression.isPending}
            onClick={() => impression.mutate()}
          >
            {impression.isPending ? 'Sending…' : 'Send test impression'}
          </button>
          {impression.error && <p>{impression.error.message}</p>}
        </div>
      </div>
    </div>
  );
}
