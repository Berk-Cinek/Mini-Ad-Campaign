export type CampaignStatus = 'active' | 'paused' | 'completed';

export interface Campaign {
  id: number;
  title: string;
  budget: number;
  spent: number;
  status: CampaignStatus;
  start_date: string;
  end_date: string;
  created_at: string;
  updated_at: string;
  remaining: number;
}

export interface Stats {
  impressions: number;
  spent: number;
  remaining: number;
  budget: number;
  status: CampaignStatus;
}

export interface CreateCampaignInput {
  title: string;
  budget: number;
  start_date: string;
  end_date: string;
}

export interface UpdateCampaignInput {
  title?: string;
  budget?: number;
  start_date?: string;
  end_date?: string;
}

export interface ApiErrorBody {
  error: string;
}
