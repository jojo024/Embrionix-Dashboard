import type { Device } from '../types/device';

// Normalize an IP address: strip CIDR prefix (/24, /30, etc), trim whitespace,
// and treat unset / 0.0.0.0 as empty.
function norm(ip?: string): string {
  const v = (ip ?? '').trim();
  if (v === '0.0.0.0' || v === '') return '';
  // Remove CIDR suffix (e.g., "10.143.225.178/30" → "10.143.225.178")
  return v.split('/')[0];
}

// ipConfigIssues returns human-readable warnings about address mismatches for a
// device, combining two checks:
//   1. Dashboard vs device — a Red/Blue management IP configured in the
//      dashboard that the device doesn't report anywhere (stale entry or the
//      device was re-addressed).
//   2. Device static vs live — a statically-configured interface whose active
//      address differs from its configured one (pending reboot / misconfig).
export function ipConfigIssues(device: Device): string[] {
  const pd = device.polling_data;
  if (!pd) return [];
  const issues: string[] = [];

  // Addresses the device actually reports (control IP + interface IPs).
  const reported = new Set<string>();
  if (norm(pd.ip_addr)) reported.add(norm(pd.ip_addr));
  for (const itf of pd.interfaces ?? []) {
    if (norm(itf.current_ip)) reported.add(norm(itf.current_ip));
    if (norm(itf.static_ip)) reported.add(norm(itf.static_ip));
  }

  // Check 1 — only meaningful once the device has reported some addressing.
  if (reported.size > 0) {
    const red = norm(device.management_ip_red);
    const blue = norm(device.management_ip_blue);
    if (red && !reported.has(red)) issues.push(`Red ${red} not reported by device`);
    if (blue && !reported.has(blue)) issues.push(`Blue ${blue} not reported by device`);
  }

  // Check 2 — static interfaces whose live address drifted from configured.
  for (const itf of pd.interfaces ?? []) {
    if (itf.dhcp) continue; // DHCP drift is expected
    const s = norm(itf.static_ip);
    const cur = norm(itf.current_ip);
    if (s && cur && s !== cur) {
      issues.push(`${itf.name}: configured ${s}, active ${cur}`);
    }
  }

  return issues;
}
