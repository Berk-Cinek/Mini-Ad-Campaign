import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router';

import { request } from '../api/client';
import { displayStatus, tabBucket } from '../lib/status';
import StatusBadge from '../components/StatusBadge';
import type { Campaign, CampaignStatus } from '../types';
import './CampaignsPage.css';

const TABS: CampaignStatus[] = ['active', 'paused', 'completed'];

type SortColumn = 'title' | 'remaining' | 'spent' | 'budget' | 'status';

function compareValue(a: Campaign, b: Campaign, column: SortColumn): number {
  switch (column) {
    case 'title':
      return a.title.localeCompare(b.title);
    case 'remaining':
      return a.remaining - b.remaining;
    case 'spent':
      return a.spent - b.spent;
    case 'budget':
      return a.budget - b.budget;
    case 'status':
      return displayStatus(a).localeCompare(displayStatus(b));
  }
}

export default function CampaignsPage() {
  const [tab, setTab] = useState<CampaignStatus>('active');
  const [sort, setSort] = useState<{ column: SortColumn; direction: 'desc' | 'asc' } | null>(null);
  const navigate = useNavigate();

  function handleHeaderClick(column: SortColumn) {
    setSort((prev) => {
      if (!prev || prev.column !== column) {
        return { column, direction: 'desc' };
      }
      if (prev.direction === 'desc') {
        return { column, direction: 'asc' };
      }
      return null;
    });
  }

  function sortIndicator(column: SortColumn) {
    if (!sort || sort.column !== column) {
      return null;
    }
    return sort.direction === 'desc' ? ' ▼' : ' ▲';
  }

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
  const sorted = sort
    ? [...visible].sort((a, b) => (sort.direction === 'desc' ? -1 : 1) * compareValue(a, b, sort.column))
    : visible;

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
              <th onClick={() => handleHeaderClick('title')}>Title{sortIndicator('title')}</th>
              <th onClick={() => handleHeaderClick('remaining')}>Remaining{sortIndicator('remaining')}</th>
              <th onClick={() => handleHeaderClick('spent')}>Spent{sortIndicator('spent')}</th>
              <th onClick={() => handleHeaderClick('budget')}>Budget{sortIndicator('budget')}</th>
              <th onClick={() => handleHeaderClick('status')}>Status{sortIndicator('status')}</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((campaign) => (
              <tr
                key={campaign.id}
                title="Details"
                onClick={() => navigate(`/campaigns/${campaign.id}`)}
              >
                <td className="title-cell" title={`${campaign.title} Details`}>{campaign.title}</td>
                <td>{campaign.remaining.toLocaleString()}</td>
                <td>{campaign.spent.toLocaleString()}</td>
                <td>{campaign.budget.toLocaleString()}</td>
                <td>
                  <StatusBadge campaign={campaign} />
                </td>
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
