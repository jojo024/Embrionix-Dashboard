interface Props {
  data: number[]
  width?: number
  height?: number
  className?: string
  stroke?: string
}

// Sparkline renders a tiny inline trend line as pure SVG — deliberately not
// recharts, so device cards stay out of the heavy chart bundle.
export function Sparkline({ data, width = 72, height = 20, className, stroke = '#38bdf8' }: Props) {
  const points = data.filter(v => Number.isFinite(v))
  if (points.length < 2) return null

  const min = Math.min(...points)
  const max = Math.max(...points)
  const range = max - min || 1
  const stepX = width / (points.length - 1)

  const d = points
    .map((v, i) => {
      const x = i * stepX
      const y = height - ((v - min) / range) * height
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
    })
    .join(' ')

  return (
    <svg width={width} height={height} className={className} viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none">
      <path d={d} fill="none" stroke={stroke} strokeWidth={1.5} strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  )
}
