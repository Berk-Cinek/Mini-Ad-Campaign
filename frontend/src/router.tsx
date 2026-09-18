import { createBrowserRouter } from 'react-router';

import Layout from './Layout';
import CampaignsPage from './pages/CampaignsPage';
import CampaignDetailPage from './pages/CampaignDetailPage';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <CampaignsPage /> },
      { path: 'campaigns/:id', element: <CampaignDetailPage /> },
    ],
  },
]);
