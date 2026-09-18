import { Link, Outlet } from 'react-router';
import './Layout.css';

export default function Layout() {
  return (
    <>
      <nav className="nav-bar">
        <Link className="brand" to="/">
          Campaigns
        </Link>
        <Link className="nav-button-primary" to="/campaigns/new">
          New Campaign
        </Link>
      </nav>
      <Outlet />
    </>
  );
}
