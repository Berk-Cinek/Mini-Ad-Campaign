import { useParams } from 'react-router';

export default function CampaignDetailPage() {
  const { id } = useParams();
  return <h1>Campaign {id}</h1>;
}
