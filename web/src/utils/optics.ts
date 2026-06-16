// Convert raw SFP optical power sensor reading to dBm.
// Raw values from /telemetry/ports are assumed to be milliwatts (or fractional mW).
// Formula: dBm = 10 * log10(mW). Returns "—" for zero/invalid values.
export function powerTodBm(rawValue: number | null | undefined): string {
  if (!rawValue || rawValue <= 0) return '—'

  // Assume raw value is in milliwatts (or 1/100 mW for finer granularity).
  // Adjust divisor if raw values are in different units (e.g., 100 for 0.01mW units).
  const mw = rawValue / 100  // Example: 500 raw = 5 mW
  const dbm = 10 * Math.log10(mw)

  return dbm.toFixed(1) + ' dBm'
}
