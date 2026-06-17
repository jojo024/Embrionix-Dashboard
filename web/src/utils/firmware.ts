// Translate Embrionix firmware version values to human-readable format
// Example: 0x5fce17b0 → 3.4.1607342000 (Dec 7, 2020)

export function parseFirmwareVersion(value?: string | null): { readable: string; timestamp?: number; date?: string } {
  if (!value) {
    return { readable: value || '—' };
  }

  // Check if it's a hex value (e.g., 0x5fce17b0)
  if (value.startsWith('0x') || value.startsWith('0X')) {
    try {
      const decimal = parseInt(value, 16);

      // The decimal value appears to be a Unix timestamp
      // Examples:
      // - 0x5fce17b0 = 1607342000 (Dec 7, 2020) → shown as 3.4.1607342000
      // - 0x5ef4b97a = 1593096570 (Jun 25, 2020) → shown as 3.1.1593096570
      //
      // The major.minor (3.4, 3.1) is not clearly derivable from the hex value alone.
      // For now, we show the timestamp. The mapping to major.minor may need to come
      // from device-reported endpoints or a lookup table.

      const date = new Date(decimal * 1000).toISOString().split('T')[0];

      return {
        readable: `${value} (${date})`,
        timestamp: decimal,
        date,
      };
    } catch (e) {
      return { readable: value };
    }
  }

  // If it's already readable (e.g., "3.4.2"), return as-is
  return { readable: value };
}
