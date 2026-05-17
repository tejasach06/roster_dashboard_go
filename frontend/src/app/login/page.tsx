'use client';

import Login from '../../views/Login';
import { PublicPage } from '../guards';

export default function Page() {
  return (
    <PublicPage>
      <Login />
    </PublicPage>
  );
}
