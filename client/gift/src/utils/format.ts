export function money(n: number, cur = '$'): string {
  const abs = Math.abs(n).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return (n < 0 ? '−' : '') + cur + abs
}

export function moneyShort(n: number, cur = '$'): string {
  const abs = Math.abs(n)
  if (abs >= 1000) return (n < 0 ? '−' : '') + cur + (abs / 1000).toFixed(1) + 'k'
  return (n < 0 ? '−' : '') + cur + abs.toFixed(0)
}

export function signed(n: number, cur = '$'): string {
  const sign = n >= 0 ? '+' : '−'
  const abs = Math.abs(n).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
  return sign + cur + abs
}

export function initialOf(name: string): string {
  if (!name) return '?'
  return name.trim()[0].toUpperCase()
}

// Stable palette used for avatar colors when no color is supplied.
const PALETTE = ['#D64933', '#2F5F4F', '#B8915A', '#4A5577', '#8B4A55', '#5B6E4A']

export function colorForId(id: string): string {
  let h = 0
  for (let i = 0; i < id.length; i++) h = (h * 31 + id.charCodeAt(i)) >>> 0
  return PALETTE[h % PALETTE.length]
}
