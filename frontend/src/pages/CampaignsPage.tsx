import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';

import { request } from '../api/client';
import { displayStatus } from '../lib/status';
import type { Campaign, CampaignStatus } from '../types';
import './CampaignsPage.css';

const TABS: CampaignStatus[] = ['active', 'paused', 'completed'];

export default function CampaignsPage() {
  const [tab, setTab] = useState<CampaignStatus>('active');

  const { data, isLoading, error } = useQuery({
    queryKey: ['campaigns'],
    queryFn: () => request<Campaign[]>('/api/campaigns'),
  });

  if (isLoading) {
    return <p>Loading…</p>;
  }

  if (error) {
    return <p>{error.message}</p>;
  }

  const campaigns = data ?? [];
  const visible = campaigns.filter((c) => c.status === tab);

  return (
    <div>
      <h1>Campaigns</h1>

      <p>
        <Link to="/campaigns/new">New Campaign</Link>
      </p>

      <div className="tabs" role="tablist">
        {TABS.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={t === tab}
            className={t === tab ? 'tab tab-active' : 'tab'}
            onClick={() => setTab(t)}
          >
            {t[0].toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      <table className="campaigns-table">
        <thead>
          <tr>
            <th>Title</th>
            <th>Budget</th>
            <th>Spent</th>
            <th>Remaining</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {visible.map((campaign) => (
            <tr key={campaign.id}>
              <td className="title-cell" title={campaign.title}>{campaign.title}</td>
              <td>{campaign.budget.toLocaleString()}</td>
              <td>{campaign.spent.toLocaleString()}</td>
              <td>{campaign.remaining.toLocaleString()}</td>
              <td>{displayStatus(campaign)}</td>
            </tr>
          ))}
          {visible.length === 0 && (
            <tr>
              <td colSpan={5}>No {tab} campaigns.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
