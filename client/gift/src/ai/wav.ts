// Browser MediaRecorder emits webm/opus; OpenAI audio-in accepts only wav or mp3.
// We decode the recorded blob via AudioContext and re-encode as 16-bit PCM WAV.

const TARGET_SAMPLE_RATE = 16000

export async function blobToWav(blob: Blob): Promise<Blob> {
  const arrayBuffer = await blob.arrayBuffer()
  const AudioCtx =
    window.AudioContext ||
    (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
  const ctx = new AudioCtx()
  try {
    const decoded = await ctx.decodeAudioData(arrayBuffer.slice(0))
    // Downsample & collapse to mono for smaller payloads.
    const mono = toMono(decoded)
    const resampled = resample(mono, decoded.sampleRate, TARGET_SAMPLE_RATE)
    return encodeWav(resampled, TARGET_SAMPLE_RATE)
  } finally {
    ctx.close().catch(() => {})
  }
}

function toMono(buffer: AudioBuffer): Float32Array {
  if (buffer.numberOfChannels === 1) return buffer.getChannelData(0)
  const n = buffer.length
  const out = new Float32Array(n)
  const channels: Float32Array[] = []
  for (let c = 0; c < buffer.numberOfChannels; c++) channels.push(buffer.getChannelData(c))
  for (let i = 0; i < n; i++) {
    let sum = 0
    for (let c = 0; c < channels.length; c++) sum += channels[c][i]
    out[i] = sum / channels.length
  }
  return out
}

function resample(samples: Float32Array, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate) return samples
  const ratio = fromRate / toRate
  const outLen = Math.floor(samples.length / ratio)
  const out = new Float32Array(outLen)
  for (let i = 0; i < outLen; i++) {
    const src = i * ratio
    const i0 = Math.floor(src)
    const i1 = Math.min(i0 + 1, samples.length - 1)
    const frac = src - i0
    out[i] = samples[i0] * (1 - frac) + samples[i1] * frac
  }
  return out
}

function encodeWav(samples: Float32Array, sampleRate: number): Blob {
  const numChannels = 1
  const bitsPerSample = 16
  const bytesPerSample = bitsPerSample / 8
  const blockAlign = numChannels * bytesPerSample
  const byteRate = sampleRate * blockAlign
  const dataSize = samples.length * bytesPerSample
  const buffer = new ArrayBuffer(44 + dataSize)
  const view = new DataView(buffer)

  let o = 0
  writeString(view, o, 'RIFF')
  o += 4
  view.setUint32(o, 36 + dataSize, true)
  o += 4
  writeString(view, o, 'WAVE')
  o += 4
  writeString(view, o, 'fmt ')
  o += 4
  view.setUint32(o, 16, true)
  o += 4
  view.setUint16(o, 1, true) // PCM
  o += 2
  view.setUint16(o, numChannels, true)
  o += 2
  view.setUint32(o, sampleRate, true)
  o += 4
  view.setUint32(o, byteRate, true)
  o += 4
  view.setUint16(o, blockAlign, true)
  o += 2
  view.setUint16(o, bitsPerSample, true)
  o += 2
  writeString(view, o, 'data')
  o += 4
  view.setUint32(o, dataSize, true)
  o += 4

  for (let i = 0; i < samples.length; i++) {
    const s = Math.max(-1, Math.min(1, samples[i]))
    view.setInt16(o, s < 0 ? s * 0x8000 : s * 0x7fff, true)
    o += 2
  }

  return new Blob([buffer], { type: 'audio/wav' })
}

function writeString(view: DataView, offset: number, str: string) {
  for (let i = 0; i < str.length; i++) view.setUint8(offset + i, str.charCodeAt(i))
}
