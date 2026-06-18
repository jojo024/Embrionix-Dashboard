import type { DevicePollingData } from '../types/device';

// Translate Embrionix firmware version values to a human-readable format.
// Example: 0x5fce17b0 → 3.4.1607342000
//
// The trailing number is the hex value converted to decimal (a Unix build
// timestamp). The major.minor prefix (e.g. "3.4") is NOT derivable from the
// hex alone, so known releases are mapped here. This is only a fallback — the
// device itself reports the readable version per firmware bank (see
// readableFirmware), which is preferred when available.
const KNOWN_VERSIONS: Record<number, string> = {
  1607342000: '3.4', // 0x5fce17b0
  1593096570: '3.1', // 0x5ef4b97a
};

export function parseFirmwareVersion(value?: string | null): { readable: string; raw?: string } {
  if (!value) {
    return { readable: '—' };
  }

  // Hex value (e.g. 0x5fce17b0) → "<major>.<minor>.<decimal>" when the release
  // is known, otherwise just the decimal build number.
  if (value.startsWith('0x') || value.startsWith('0X')) {
    const decimal = parseInt(value, 16);
    if (!Number.isNaN(decimal)) {
      const prefix = KNOWN_VERSIONS[decimal];
      return {
        readable: prefix ? `${prefix}.${decimal}` : `${decimal}`,
        raw: value,
      };
    }
    return { readable: value, raw: value };
  }

  // Already human-readable (e.g. "3.4.2") — return as-is.
  return { readable: value, raw: value };
}

// A readable firmware version looks like "3.4.1607342000" — leading major.minor
// digits. Hex build ids ("0x…") and product descriptions don't match.
function looksLikeVersion(s?: string): boolean {
  return !!s && /^\d+\.\d+/.test(s);
}

// readableFirmware resolves the best human-readable firmware version for a
// device. The device reports a readable version per firmware bank; the running
// firmware is the active bank, so its version (or description, whichever holds
// the human-readable string) is preferred. Falls back to the
// /self/information current_version (often a hex build id) translated via
// parseFirmwareVersion.
export function readableFirmware(pd?: DevicePollingData | null, fallback?: string | null): string {
  const active = pd?.firmware_slots?.find(s => s.active);
  if (active) {
    if (looksLikeVersion(active.version)) return active.version;
    if (looksLikeVersion(active.desc)) return active.desc;
  }
  return parseFirmwareVersion(active?.version || pd?.current_version || fallback).readable;
}
