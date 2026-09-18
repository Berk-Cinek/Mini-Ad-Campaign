import type { Campaign, CampaignStatus } from '../types';

// Display-only labels layered on top of the stored status — never written
// back to the API. "Scheduled"/"Ended" reflect the campaign's dates even
// though the stored status hasn't (yet) caught up (e.g. the background job
// that marks past-end-date campaigns completed runs once a minute).
export type DisplayStatus = 'Scheduled' | 'Ended' | CampaignStatus;

export function displayStatus(campaign: Campaign): DisplayStatus {
  const now = new Date();

  if (campaign.status === 'active' && new Date(campaign.start_date) > now) {
    return 'Scheduled';
  }

  if (
    (campaign.status === 'active' || campaign.status === 'paused') &&
    new Date(campaign.end_date) <= now
  ) {
    return 'Ended';
  }

  return campaign.status;
}
