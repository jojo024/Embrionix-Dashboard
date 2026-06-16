// Convert raw SFP optical power sensor reading to dBm.
// Raw values from /telemetry/ports are in µW (microwatts).
// Formula: dBm = 10 * log10(µW / 1000)
export function powerTodBm(rawValue: number | null | undefined): string {
  if (!rawValue || rawValue <= 0) return '—'

  // Raw value is in µW; convert to mW then to dBm
  // 353 µW = 0.353 mW = 10*log10(0.353) = -4.52 dBm
  const mw = rawValue / 1000
  const dbm = 10 * Math.log10(mw)

  return dbm.toFixed(2) + ' dBm'
}
