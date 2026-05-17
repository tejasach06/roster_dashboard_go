'use client';

import Dashboard from '../views/Dashboard';
import { PrivatePage } from './guards';

export default function Page() {
  return (
    <PrivatePage>
      <Dashboard />
    </PrivatePage>
  );
}
