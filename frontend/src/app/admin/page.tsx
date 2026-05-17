'use client';

import AdminPanel from '../../views/AdminPanel';
import { PrivatePage } from '../guards';

export default function Page() {
  return (
    <PrivatePage adminOnly>
      <AdminPanel />
    </PrivatePage>
  );
}
