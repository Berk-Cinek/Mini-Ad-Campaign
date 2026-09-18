import { displayStatus } from '../lib/status';
import type { Campaign } from '../types';
import type { DisplayStatus } from '../lib/status';
import './StatusBadge.css';

const BADGE: Record<DisplayStatus, { label: string; className: string }> = {
  Scheduled: { label: 'Scheduled', className: 'badge-scheduled' },
  active: { label: 'Active', className: 'badge-active' },
  paused: { label: 'Paused', className: 'badge-paused' },
  Ended: { label: 'Ended', className: 'badge-ended' },
  completed: { label: 'Completed', className: 'badge-completed' },
};

export default function StatusBadge({ campaign }: { campaign: Campaign }) {
  const { label, className } = BADGE[displayStatus(campaign)];
  return <span className={`badge ${className}`}>{label}</span>;
}
