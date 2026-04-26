# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project shape

A Tauri 2 desktop+mobile shell wrapping a Vue 3 + TypeScript + Vite SPA. The client talks to a self-hosted "gift" backend (a personal/group expense tracker — spendings, incomes, budgets, goals, groups, settings). Voice input is parsed by an OpenAI-compatible LLM directly from the browser.

`bun` is the package manager (see `bun.lock`); Tauri is wired to `bun run dev` / `bun run build`.

## Commands

- `bun run dev` — Vite dev server on **port 1420** (fixed; `strictPort: true` so port conflicts are fatal).
- `bun run build` — `vue-tsc --noEmit` then `vite build`. There is no separate lint/test step; the typecheck is the gate.
- `bun run preview` — preview the built `dist/`.
- `bun run tauri dev` / `bun run tauri build` — desktop/mobile builds. Tauri invokes `bun run dev`/`build` itself via `beforeDevCommand` / `beforeBuildCommand` in `src-tauri/tauri.conf.json`. Don't start `vite` separately when using `tauri dev`.
- `bun run tauri android dev` / `bun run tauri ios dev` — mobile (generated projects under `src-tauri/gen/`). Set `TAURI_DEV_HOST` so Vite binds to a LAN address; `vite.config.ts` already wires HMR to port 1421 in that case.
- `ALLOWED_HOSTS=host1,host2 bun run dev` — needed if accessing the dev server through a tunnel/proxy.

There is no test suite. Don't invent one.

## Architecture

### Runtime configuration, not env vars

There are no `.env` files and no `VITE_*` build-time config. **Everything user-facing is configured at runtime and lives in `localStorage`:**

- `gift.server_url` — backend base URL, set via `ServerSetupView` (`src/stores/server.ts`). Read with `${baseUrl}/api/v1` prefix in `src/api/client.ts`.
- `gift.access_token` / `gift.refresh_token` / `gift.user` — auth state (`src/stores/auth.ts`).
- `gift.llm_config` — `{base_url, api_key, chat_model}` for the OpenAI-compatible LLM used by voice input (`src/stores/llm.ts`). Default model is `gpt-4o-audio-preview`.
- `gift.user_profile`, `gift.locale` — profile cache and i18n preference.

This means the app boots into a three-stage gate: **server configured → authenticated → ready**. The router guard in `src/router/index.ts` enforces this order; new routes need `meta: { requiresAuth: true }` (or `guest`/`setup`) for the guard to handle them.

### State: handcrafted ref-stores, not Pinia/Vuex

Each file in `src/stores/` exports a plain object holding `ref`s and methods (`auth`, `server`, `user`, `llm`, `toast`). They hydrate from `localStorage` at module load and write back on mutation. Treat them as singletons — there is no provide/inject. When adding persistent client state, follow the same pattern (load/persist helpers + a single `export const xxx = { ... }`).

### API layer (`src/api/`)

`client.ts` is the only place that calls `fetch` for JSON endpoints. Every backend response is the envelope `{ status, message, data }`; `parseWrapped` returns `data` on success or throws `ApiError(message, status)` on failure. **Don't bypass `request()` / `api.{get,post,put,delete}` for normal endpoints** — you'd lose the unified error shape and the automatic single-shot refresh on 401 (`refreshOnce()` calls `/auth/refresh` and retries once; on failure clears auth).

Binary/non-wrapped endpoints (the only current example is `settingsApi.export` which downloads a file) intentionally use `fetch` directly. If you add another such endpoint, follow that pattern and keep the auth header logic explicit.

`endpoints.ts` groups calls per resource (`authApi`, `userApi`, `groupApi`, `spendingApi`, `incomeApi`, `budgetApi`, `goalApi`, `settingsApi`). Type definitions for request/response shapes live in `types.ts`. Backend resource IDs may appear as either `id` or `_id` — types reflect this; handle both when reading.

### Voice input pipeline

The flow is: `useVoiceInput` (`src/composables/`) records via `MediaRecorder` → `blobToWav` (`src/ai/wav.ts`) re-encodes to WAV → a parser from `src/ai/parse.ts` sends it to the LLM.

`parse.ts` uses the Vercel AI SDK with `@ai-sdk/openai` and **forces a single tool call** (`toolChoice: { type: 'tool', toolName }`). The tool's `inputSchema` is a Zod schema; the parsed fields come from `result.toolCalls[0].input` — there is no JSON parse round-trip. The provider is invoked through `provider.chat(model)` (Chat Completions, `/chat/completions`) on purpose: the default Responses API path (`/responses`) is not implemented by OpenRouter and rejects audio file parts in this SDK version. Keep this when extending.

When adding a new "speak to fill a form" feature, add a `*DraftSchema` + a `parseXFromAudio` wrapper alongside the existing ones rather than calling the AI SDK from a component.

### i18n

`src/i18n/index.ts` exports `t(key, params?)` and `useI18n()`. Three locales (`en`, `ru`, `uz`) live as flat key→string maps in `src/i18n/locales/`. The `t` function reads `locale.value` so it's reactive in templates. **Add new strings to all three locale files**; missing keys silently fall back to English, then to the raw key.

### Tauri side (`src-tauri/`)

Minimal Rust — only `tauri-plugin-opener` is wired up and a placeholder `greet` command in `src-tauri/src/lib.rs`. Capabilities are in `src-tauri/capabilities/`. The mobile projects are generated into `src-tauri/gen/{android,apple}` and are checked in. `src-tauri/gen/android/keystore.properties` is gitignored-by-convention (currently untracked) — never commit signing material.

CSP is set to `null` in `tauri.conf.json` because the LLM and backend hosts are user-supplied at runtime; tightening this would break the runtime-configurable architecture.
