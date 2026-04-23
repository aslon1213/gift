import { ref } from 'vue'

const message = ref<string | null>(null)
let timer: ReturnType<typeof setTimeout> | null = null

function flash(msg: string, ms = 2400) {
  if (timer) clearTimeout(timer)
  message.value = msg
  timer = setTimeout(() => {
    message.value = null
    timer = null
  }, ms)
}

function clear() {
  if (timer) clearTimeout(timer)
  timer = null
  message.value = null
}

export const toast = {
  message,
  flash,
  clear,
}
