import type { CurrencyCode } from '../api/types'

// Per-currency presentation rules. `suffix` means the symbol comes AFTER the number.
export const CURRENCY_META: Record<
  CurrencyCode,
  { symbol: string; position: 'prefix' | 'suffix'; spacing: boolean; decimals: 0 | 2 }
> = {
  USD: { symbol: '$', position: 'prefix', spacing: false, decimals: 2 },
  EUR: { symbol: '€', position: 'prefix', spacing: false, decimals: 2 },
  // UZS rarely uses decimals in practice — soʻm is written after, with a space.
  UZS: { symbol: "so'm", position: 'suffix', spacing: true, decimals: 0 },
}

export const CURRENCY_CODES: CurrencyCode[] = ['USD', 'EUR', 'UZS']

// Accepts either a currency CODE ('USD'/'EUR'/'UZS') or a legacy raw symbol ('$').
// Raw symbols are passed through and treated as prefix/no-space/2-decimals.
function resolveMeta(
  currency: CurrencyCode | string = 'USD',
): { symbol: string; position: 'prefix' | 'suffix'; spacing: boolean; decimals: 0 | 2 } {
  if (currency in CURRENCY_META) {
    return CURRENCY_META[currency as CurrencyCode]
  }
  return { symbol: currency, position: 'prefix', spacing: false, decimals: 2 }
}

function fmtAbs(n: number, decimals: 0 | 2): string {
  return Math.abs(n).toLocaleString('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}

function join(sign: string, body: string, meta: ReturnType<typeof resolveMeta>): string {
  const sep = meta.spacing ? ' ' : ''
  if (meta.position === 'prefix') return sign + meta.symbol + sep + body
  return sign + body + sep + meta.symbol
}

export function money(n: number, currency: CurrencyCode | string = 'USD'): string {
  const meta = resolveMeta(currency)
  const sign = n < 0 ? '−' : ''
  return join(sign, fmtAbs(n, meta.decimals), meta)
}

export function signed(n: number, currency: CurrencyCode | string = 'USD'): string {
  const meta = resolveMeta(currency)
  const sign = n >= 0 ? '+' : '−'
  return join(sign, fmtAbs(n, meta.decimals), meta)
}

export function moneyWhole(n: number, currency: CurrencyCode | string = 'USD'): string {
  const meta = resolveMeta(currency)
  const sign = n < 0 ? '−' : ''
  return join(sign, Math.abs(Math.round(n)).toLocaleString('en-US'), meta)
}

export function signedWhole(n: number, currency: CurrencyCode | string = 'USD'): string {
  const meta = resolveMeta(currency)
  const sign = n >= 0 ? '+' : '−'
  return join(sign, Math.abs(Math.round(n)).toLocaleString('en-US'), meta)
}

// Compact: "$12.8k" / "1.2M soʻm" — nice on narrow cards.
export function moneyShort(n: number, currency: CurrencyCode | string = 'USD'): string {
  const meta = resolveMeta(currency)
  const abs = Math.abs(n)
  const sign = n < 0 ? '−' : ''
  let body: string
  if (abs >= 1_000_000) body = (abs / 1_000_000).toFixed(1) + 'M'
  else if (abs >= 1_000) body = (abs / 1_000).toFixed(1) + 'k'
  else body = abs.toFixed(0)
  return join(sign, body, meta)
}

export function currencySymbol(currency: CurrencyCode | string = 'USD'): string {
  return resolveMeta(currency).symbol
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
