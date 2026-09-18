import { useState } from 'react';
import type { FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';

import { request } from '../api/client';
import { toDatetimeLocalValue } from '../lib/datetime';
import type { Campaign, CreateCampaignInput } from '../types';
import './NewCampaignPage.css';

export default function NewCampaignPage() {
  const [title, setTitle] = useState('');
  const [budget, setBudget] = useState('');
  const [startDate, setStartDate] = useState(() => toDatetimeLocalValue(new Date()));
  const [endDate, setEndDate] = useState('');

  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const mutation = useMutation({
    mutationFn: (input: CreateCampaignInput) =>
      request<Campaign>('/api/campaigns', { method: 'POST', body: input }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['campaigns'] });
      navigate('/');
    },
  });

  function handleSubmit(e: FormEvent) {
    e.preventDefault();

    if (!title.trim() || !budget || !startDate || !endDate) {
      return;
    }

    mutation.mutate({
      title,
      budget: Number(budget),
      start_date: new Date(startDate).toISOString(),
      end_date: new Date(endDate).toISOString(),
    });
  }

  return (
    <div>
      <h1>New Campaign</h1>

      <div className="form-card">
        <form className="campaign-form" onSubmit={handleSubmit}>
          <div className="form-fields">
            <label>
              Title
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                required
                maxLength={200}
              />
            </label>

            <label>
              Budget
              <input
                type="number"
                value={budget}
                onChange={(e) => setBudget(e.target.value)}
                required
                min={1}
                step={1}
              />
            </label>

            <label>
              Start date
              <input
                type="datetime-local"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                required
              />
            </label>

            <label>
              End date
              <input
                type="datetime-local"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                required
              />
            </label>

            {mutation.error && <p className="form-error">{mutation.error.message}</p>}
          </div>

          <div className="form-actions">
            <button className="btn-secondary" type="button" onClick={() => navigate('/')}>
              Cancel
            </button>
            <button className="btn-primary" type="submit" disabled={mutation.isPending}>
              {mutation.isPending ? 'Creating…' : 'Create campaign'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
