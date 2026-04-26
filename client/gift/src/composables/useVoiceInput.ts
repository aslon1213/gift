import { onUnmounted, ref } from 'vue'
import { blobToWav } from '../ai/wav'

export type VoiceStatus = 'idle' | 'recording' | 'thinking' | 'error'

function pickMimeType(): string {
  const candidates = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg']
  for (const type of candidates) {
    if (typeof MediaRecorder !== 'undefined' && MediaRecorder.isTypeSupported(type)) return type
  }
  return ''
}

export function useVoiceInput() {
  const status = ref<VoiceStatus>('idle')
  const error = ref<string | null>(null)

  let recorder: MediaRecorder | null = null
  let chunks: Blob[] = []
  let stream: MediaStream | null = null
  let resolveStop: ((blob: Blob) => void) | null = null

  async function start() {
    if (status.value !== 'idle' && status.value !== 'error') return
    error.value = null
    if (!navigator.mediaDevices?.getUserMedia) {
      error.value = 'Microphone access not available on this device'
      status.value = 'error'
      return
    }
    try {
      stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mimeType = pickMimeType()
      recorder = new MediaRecorder(stream, mimeType ? { mimeType } : undefined)
      chunks = []
      recorder.ondataavailable = (e) => {
        if (e.data && e.data.size) chunks.push(e.data)
      }
      recorder.onstop = () => {
        const blob = new Blob(chunks, { type: recorder?.mimeType || 'audio/webm' })
        resolveStop?.(blob)
        resolveStop = null
        stream?.getTracks().forEach((t) => t.stop())
        stream = null
        recorder = null
      }
      recorder.start()
      status.value = 'recording'
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to start recording'
      status.value = 'error'
    }
  }

  function stop(): Promise<Blob | null> {
    if (!recorder || recorder.state === 'inactive') return Promise.resolve(null)
    return new Promise<Blob | null>((resolve) => {
      resolveStop = (blob) => resolve(blob)
      try {
        recorder?.stop()
      } catch {
        resolve(null)
      }
    })
  }

  function cancel() {
    try {
      recorder?.stop()
    } catch {
      // ignore
    }
    stream?.getTracks().forEach((t) => t.stop())
    stream = null
    recorder = null
    chunks = []
    resolveStop = null
    status.value = 'idle'
  }

  // Stop recording, convert to WAV, then hand off to the caller's parser.
  async function run<T>(parser: (audio: Blob) => Promise<T>): Promise<T | null> {
    if (status.value !== 'recording') return null
    const raw = await stop()
    if (!raw) {
      status.value = 'idle'
      return null
    }
    status.value = 'thinking'
    try {
      const wav = await blobToWav(raw)
      const result = await parser(wav)
      status.value = 'idle'
      return result
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Voice parsing failed'
      status.value = 'error'
      return null
    }
  }

  function reset() {
    status.value = 'idle'
    error.value = null
  }

  onUnmounted(() => {
    cancel()
  })

  return { status, error, start, stop, cancel, run, reset }
}
