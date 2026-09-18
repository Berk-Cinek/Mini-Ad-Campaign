import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';

import { request } from '../api/client';
import { tabBucket } from '../lib/status';
import StatusBadge from '../components/StatusBadge';
import type { Campaign, CampaignStatus } from '../types';
import './CampaignsPage.css';

const TABS: CampaignStatus[] = ['active', 'paused', 'completed'];

export default function CampaignsPage() {
  const [tab, setTab] = useState<CampaignStatus>('active');
  const navigate = useNavigate();

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
  const visible = campaigns.filter((c) => tabBucket(c) === tab);

  return (
    <div>
      <h1>AdManager</h1>

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

      <div className="table-card">
        <table className="campaigns-table">
          <thead>
            <tr>
              <th>Title</th>
              <th>Status</th>
              <th>Budget</th>
              <th>Spent</th>
              <th>Remaining</th>
            </tr>
          </thead>
          <tbody>
            {visible.map((campaign) => (
              <tr
                key={campaign.id}
                title="Details"
                onClick={() => navigate(`/campaigns/${campaign.id}`)}
              >
                <td className="title-cell" title={`${campaign.title} Details`}>{campaign.title}</td>
                <td>
                  <StatusBadge campaign={campaign} />
                </td>
                <td>{campaign.budget.toLocaleString()}</td>
                <td>{campaign.spent.toLocaleString()}</td>
                <td>{campaign.remaining.toLocaleString()}</td>
              </tr>
            ))}
            {visible.length === 0 && (
              <tr className="empty-row">
                <td colSpan={5}>No {tab} campaigns.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
