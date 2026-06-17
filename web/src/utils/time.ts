export function formatRelativeTime(iso: string): string {
  const date = new Date(iso);
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000);

  // Handle future timestamps (shouldn't happen, but safety check)
  if (seconds < 0) return 'just now';

  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return date.toLocaleDateString();
}

export function formatDate(iso: string): string {
  return new Date(iso).toLocaleString();
}
