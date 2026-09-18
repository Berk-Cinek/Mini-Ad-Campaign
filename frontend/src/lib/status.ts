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

// Which tab a campaign belongs in. Differs from displayStatus() only for
// "Ended": that's grouped under Completed (it's functionally done, just
// waiting on the once-a-minute background job), but the Status cell still
// shows the more specific "Ended" label via displayStatus() — this only
// changes which tab a row lands in, not what it displays.
export function tabBucket(campaign: Campaign): CampaignStatus {
  return displayStatus(campaign) === 'Ended' ? 'completed' : campaign.status;
}
