import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router';

import { request } from '../api/client';
import StatusBadge from '../components/StatusBadge';
import type { Campaign, Stats } from '../types';

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

  function invalidateDetail() {
    queryClient.invalidateQueries({ queryKey: ['campaign', id] });
    queryClient.invalidateQueries({ queryKey: ['stats', id] });
  }

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

  return (
    <div>
      <h1>{campaign.title}</h1>

      <section>
        <p>
          Status: <StatusBadge campaign={campaign} />
        </p>
        <p>Budget: {campaign.budget.toLocaleString()}</p>
        <p>Start date: {new Date(campaign.start_date).toLocaleString()}</p>
        <p>End date: {new Date(campaign.end_date).toLocaleString()}</p>
      </section>

      <section>
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

      <section>
        <h2>Actions</h2>

        <div>
          <button type="button" disabled={isCompleted || pause.isPending} onClick={() => pause.mutate()}>
            {pause.isPending ? 'Pausing…' : 'Pause'}
          </button>
          {pause.error && <p>{pause.error.message}</p>}
        </div>

        <div>
          <button type="button" disabled={isCompleted || resume.isPending} onClick={() => resume.mutate()}>
            {resume.isPending ? 'Resuming…' : 'Resume'}
          </button>
          {resume.error && <p>{resume.error.message}</p>}
        </div>

        <div>
          <button type="button" disabled={isCompleted || end.isPending} onClick={() => end.mutate()}>
            {end.isPending ? 'Ending…' : 'End'}
          </button>
          {end.error && <p>{end.error.message}</p>}
        </div>

        <div>
          <button type="button" disabled={del.isPending} onClick={() => del.mutate()}>
            {del.isPending ? 'Deleting…' : 'Delete'}
          </button>
          {del.error && <p>{del.error.message}</p>}
        </div>
      </section>

      <section>
        <h2>Debug / Simulation</h2>
        <p>
          In production, impressions are recorded by the ad server, not by hand. This
          demonstrates budget deduction and auto-pause.
        </p>
        <button type="button" disabled={impression.isPending} onClick={() => impression.mutate()}>
          {impression.isPending ? 'Sending…' : 'Send test impression'}
        </button>
        {impression.error && <p>{impression.error.message}</p>}
      </section>
    </div>
  );
}
