import { clsx } from 'clsx';
import type { DeviceStatus } from '../types/device';

const LABELS: Record<DeviceStatus, string> = {
  online: 'Online',
  offline: 'Offline',
  warning: 'Warning',
  critical: 'Critical',
  unknown: 'Unknown',
};

interface Props {
  status: DeviceStatus;
  showDot?: boolean;
  size?: 'sm' | 'md';
}

export function StatusBadge({ status, showDot = true, size = 'sm' }: Props) {
  return (
    <span
      className={clsx(
        'inline-flex items-center gap-1.5 rounded-full font-medium',
        size === 'sm' ? 'px-2 py-0.5 text-xs' : 'px-2.5 py-1 text-sm',
        `badge-${status}`,
      )}
    >
      {showDot && <span className={clsx('status-dot', `status-${status}`)} />}
      {LABELS[status]}
    </span>
  );
}
