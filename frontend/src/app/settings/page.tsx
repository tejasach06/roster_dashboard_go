'use client';

import Settings from '../../views/Settings';
import { PrivatePage } from '../guards';

export default function Page() {
  return (
    <PrivatePage adminOnly>
      <Settings />
    </PrivatePage>
  );
}
