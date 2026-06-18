// Translate Embrionix firmware version values to a human-readable format.
// Example: 0x5fce17b0 → 3.4.1607342000
//
// The trailing number is the hex value converted to decimal (a Unix build
// timestamp). The major.minor prefix (e.g. "3.4") is NOT derivable from the
// hex alone, so known releases are mapped here. Add new releases as they ship.
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
