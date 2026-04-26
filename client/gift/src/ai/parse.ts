import { generateText, tool } from 'ai'
import { createOpenAI } from '@ai-sdk/openai'
import { z } from 'zod'
import { llm } from '../stores/llm'

export const SpendingDraftSchema = z.object({
  amount: z
    .number()
    .nullable()
    .describe('Numeric spending amount in major units (e.g. "5 million" → 5000000). Null if not mentioned.'),
  currency: z
    .string()
    .nullable()
    .describe('ISO currency code like USD, EUR, RUB, UZS. Null if not mentioned.'),
  category: z
    .enum(['Food', 'Travel', 'Stay', 'Transport', 'Activity', 'Groceries', 'Coffee', 'Home'])
    .nullable()
    .describe(
      'ONLY fill if the user explicitly names a category (e.g. "category food", "put it under travel"). ' +
        'Do NOT infer a category from descriptive nouns like "restaurant", "uber", "groceries store" — ' +
        'those go to description. When unsure, return null.',
    ),
  description: z
    .string()
    .nullable()
    .describe(
      'Short free-text label for the spending (e.g. "Restaurant", "Uber to airport", "Locavore dinner"). ' +
        'This is where descriptive nouns belong, NOT in category. Null only if the user said nothing descriptive.',
    ),
  date: z
    .string()
    .nullable()
    .describe(
      'ISO date (YYYY-MM-DD). Resolve relative words like "today", "yesterday", "last Friday" against the ' +
        '"Today is" reference in the system prompt. If the user does not mention a date at all, return null.',
    ),
})
export type SpendingDraft = z.infer<typeof SpendingDraftSchema>

export const BudgetDraftSchema = z.object({
  category: z.string().nullable().describe('Budget category, e.g. "Food". Null if unclear.'),
  limit: z
    .number()
    .nullable()
    .describe('Budget cap (the maximum allowed). Most spoken phrasings — "budget 500 a month for food" — mean this. Null if not mentioned.'),
  amount: z
    .number()
    .nullable()
    .describe('Already-spent amount, only if the user explicitly states a starting balance ("already spent 200"). Null otherwise.'),
  currency: z.string().nullable().describe('ISO currency code. Null if not mentioned.'),
  period: z
    .enum(['weekly', 'monthly', 'trip', 'yearly'])
    .nullable()
    .describe('Budget period. Null if unclear.'),
})
export type BudgetDraft = z.infer<typeof BudgetDraftSchema>

export const GoalDraftSchema = z.object({
  name: z.string().nullable().describe('Short goal name, e.g. "Bali trip". Null if unclear.'),
  target_amount: z.number().nullable().describe('Target amount to save. Null if not mentioned.'),
  current_amount: z.number().nullable().describe('Already saved amount. Null if not mentioned.'),
  currency: z.string().nullable().describe('ISO currency code. Null if not mentioned.'),
  deadline: z.string().nullable().describe('ISO date (YYYY-MM-DD). Null if not mentioned.'),
})
export type GoalDraft = z.infer<typeof GoalDraftSchema>

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

function client() {
  const cfg = llm.config.value
  if (!cfg.base_url || !cfg.api_key || !cfg.chat_model) {
    throw new Error('LLM provider not configured. Open Settings → Voice & AI.')
  }
  const openai = createOpenAI({ baseURL: cfg.base_url, apiKey: cfg.api_key })
  return { provider: openai, model: cfg.chat_model }
}

async function audioBytes(blob: Blob): Promise<Uint8Array> {
  const buf = await blob.arrayBuffer()
  return new Uint8Array(buf)
}

// Generic audio → single-tool-call extraction. The model listens to the audio
// and MUST call the given tool with the extracted fields. We read the input
// straight off toolCalls[0] — no structured-output JSON parsing round-trip.
async function extractFromAudio<T>(
  audio: Blob,
  toolName: string,
  toolDescription: string,
  schema: z.ZodType<T>,
  instruction: string,
): Promise<T> {
  const { provider, model } = client()
  const data = await audioBytes(audio)

  const tools = {
    [toolName]: tool({
      description: toolDescription,
      inputSchema: schema,
    }),
  }

  const result = await generateText({
    // Force Chat Completions API (/chat/completions). The provider's default
    // routes through the Responses API (/responses), which OpenRouter doesn't
    // implement and which rejects audio file parts in this SDK version.
    model: provider.chat(model),
    tools,
    toolChoice: { type: 'tool', toolName },
    messages: [
      {
        role: 'user',
        content: [
          { type: 'text', text: instruction },
          { type: 'file', data, mediaType: audio.type || 'audio/wav' },
        ],
      },
    ],
  })

  const call = result.toolCalls.find((c) => c.toolName === toolName)
  if (!call) throw new Error('Model did not call the extraction tool')
  return call.input as T
}

export function parseSpendingFromAudio(audio: Blob): Promise<SpendingDraft> {
  const now = new Date()
  const iso = now.toISOString().slice(0, 10)
  const human = now.toLocaleDateString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
  return extractFromAudio<SpendingDraft>(
    audio,
    'record_spending',
    'Record a single spending/expense from the user’s spoken message.',
    SpendingDraftSchema,
    `Listen to the audio and call record_spending with ONLY the fields the speaker actually mentions. ` +
      `Leave anything not stated as null — do not guess. ` +
      `Specifically: never put descriptive words ("restaurant", "uber", "coffee") into the category field; ` +
      `those belong in description. Only set category when the speaker literally names one of the allowed values. ` +
      `For dates, today is ${human} (${iso}). Resolve "today", "yesterday", "last Monday", etc. against this reference and emit YYYY-MM-DD.`,
  )
}

export function parseBudgetFromAudio(audio: Blob): Promise<BudgetDraft> {
  return extractFromAudio<BudgetDraft>(
    audio,
    'record_budget',
    'Record a category budget from the user’s spoken message.',
    BudgetDraftSchema,
    `Listen to the audio and call record_budget with the fields it describes. ` +
      `Leave anything not explicitly stated as null.`,
  )
}

export function parseGoalFromAudio(audio: Blob): Promise<GoalDraft> {
  return extractFromAudio<GoalDraft>(
    audio,
    'record_goal',
    'Record a savings goal from the user’s spoken message.',
    GoalDraftSchema,
    `Listen to the audio and call record_goal with the fields it describes. ` +
      `Leave anything not explicitly stated as null. Today is ${today()}.`,
  )
}
