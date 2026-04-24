import { computed, ref } from 'vue'
import en from './locales/en'
import ru from './locales/ru'
import uz from './locales/uz'

export type Locale = 'en' | 'ru' | 'uz'
export const SUPPORTED_LOCALES: Locale[] = ['en', 'ru', 'uz']

const DICTS: Record<Locale, Record<string, string>> = { en, ru, uz }

const STORAGE_KEY = 'gift.locale'

function initialLocale(): Locale {
  const saved = localStorage.getItem(STORAGE_KEY) as Locale | null
  if (saved && SUPPORTED_LOCALES.includes(saved)) return saved
  // Fall back to the first supported browser language
  const nav = (navigator.language || 'en').slice(0, 2).toLowerCase() as Locale
  return SUPPORTED_LOCALES.includes(nav) ? nav : 'en'
}

const locale = ref<Locale>(initialLocale())

function setLocale(next: Locale) {
  if (!SUPPORTED_LOCALES.includes(next)) return
  locale.value = next
  localStorage.setItem(STORAGE_KEY, next)
  document.documentElement.setAttribute('lang', next)
}

// Apply once on boot
document.documentElement.setAttribute('lang', locale.value)

function interpolate(template: string, params?: Record<string, string | number>): string {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_, key) =>
    key in params ? String(params[key]) : `{${key}}`,
  )
}

// Reactive translator. Reading `locale.value` inside triggers reactivity
// when called from a template expression.
export function t(key: string, params?: Record<string, string | number>): string {
  const dict = DICTS[locale.value] ?? DICTS.en
  const raw = dict[key] ?? DICTS.en[key] ?? key
  return interpolate(raw, params)
}

export const LOCALE_META: Record<Locale, { label: string; short: string }> = {
  en: { label: 'English', short: 'EN' },
  ru: { label: 'Русский', short: 'RU' },
  uz: { label: "O'zbekcha", short: 'UZ' },
}

export const i18n = {
  locale: computed(() => locale.value),
  setLocale,
  t,
}

// Convenience composable for SFCs that prefer a named import.
export function useI18n() {
  return { t, locale: computed(() => locale.value), setLocale }
}
