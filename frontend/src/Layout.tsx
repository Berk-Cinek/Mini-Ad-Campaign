import { Link, Outlet } from 'react-router';

export default function Layout() {
  return (
    <>
      <nav>
        <Link to="/">Campaigns</Link>
      </nav>
      <Outlet />
    </>
  );
}
