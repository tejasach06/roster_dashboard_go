'use client';

import TeamRoster from '../../../views/TeamRoster';
import { PrivatePage } from '../../guards';

export default function Page() {
  return (
    <PrivatePage>
      <TeamRoster />
    </PrivatePage>
  );
}
