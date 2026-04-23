const PALETTE = [
  '#a78bfa', '#60a5fa', '#34d399', '#fbbf24', '#f472b6',
  '#f87171', '#22d3ee', '#c084fc', '#fb923c', '#4ade80',
]

export function colorFor(i: number): string {
  return PALETTE[i % PALETTE.length]
}

export function groupBy<T>(items: T[], key: (t: T) => string): Map<string, T[]> {
  const out = new Map<string, T[]>()
  for (const item of items) {
    const k = key(item)
    const bucket = out.get(k)
    if (bucket) bucket.push(item)
    else out.set(k, [item])
  }
  return out
}

export function sumBy<T>(items: T[], value: (t: T) => number): number {
  let total = 0
  for (const item of items) total += value(item)
  return total
}

export function formatDay(d: Date): string {
  return d.toISOString().slice(0, 10)
}

export function lastNDays(n: number): string[] {
  const out: string[] = []
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  for (let i = n - 1; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(d.getDate() - i)
    out.push(formatDay(d))
  }
  return out
}
