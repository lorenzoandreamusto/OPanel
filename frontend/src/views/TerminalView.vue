<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-white">Web Terminal</h2>
      <div class="flex items-center gap-3">
        <span :class="connected ? 'badge-success' : 'badge-danger'" class="badge">
          {{ connected ? 'Connected' : 'Disconnected' }}
        </span>
        <button @click="toggleConnection" class="btn-primary text-sm">
          {{ connected ? 'Disconnect' : 'Connect' }}
        </button>
      </div>
    </div>

    <div class="card p-0 overflow-hidden">
      <div ref="terminalEl"
           class="bg-black p-4 font-mono text-sm text-green-400 overflow-auto"
           style="height: 500px; white-space: pre-wrap; word-break: break-all;"
           tabindex="0"
           @keydown="handleKeydown"
      >
        <div v-for="(line, i) in outputLines" :key="i">{{ line }}</div>
        <span class="text-green-400">{{ prompt }}</span><span class="text-white">{{ currentInput }}</span><span class="animate-pulse">_</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

const terminalEl = ref<HTMLElement | null>(null)
const outputLines = ref<string[]>(['Connecting...'])
const currentInput = ref('')
const prompt = ref('$ ')
const connected = ref(false)
let ws: WebSocket | null = null
let currentLine = ''

function connect() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('opanel_token')
  ws = new WebSocket(`${protocol}//${window.location.host}/api/terminal/ws?token=${token}`)

  ws.onopen = () => {
    connected.value = true
    outputLines.value = ['Connected to terminal']
    ws?.send(JSON.stringify({ type: 'resize', width: 120, height: 30 }))
  }

  ws.onclose = () => {
    connected.value = false
    outputLines.value.push('Disconnected')
  }

  ws.onerror = () => {
    connected.value = false
  }

  ws.onmessage = (event) => {
    const data = event.data
    if (typeof data === 'string') {
      const clean = data.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '')
      if (clean === '\r\n' || clean === '\n') {
        outputLines.value.push(currentLine)
        currentLine = ''
      } else if (clean === '\r') {
        currentLine = ''
      } else {
        currentLine += clean
        if (clean.includes('$ ') || clean.includes('# ')) {
          const parts = currentLine.split('\n')
          for (const part of parts) {
            if (part.trim()) outputLines.value.push(part)
          }
          currentLine = ''
        }
      }
      if (outputLines.value.length > 500) {
        outputLines.value = outputLines.value.slice(-300)
      }
      scrollToBottom()
    }
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (terminalEl.value) {
      terminalEl.value.scrollTop = terminalEl.value.scrollHeight
    }
  })
}

function handleKeydown(e: KeyboardEvent) {
  if (!connected.value || !ws) return

  e.preventDefault()

  if (e.key === 'Enter') {
    ws.send(currentInput.value + '\n')
    outputLines.value.push(prompt.value + currentInput.value)
    currentInput.value = ''
  } else if (e.key === 'Backspace') {
    currentInput.value = currentInput.value.slice(0, -1)
  } else if (e.key.length === 1) {
    currentInput.value += e.key
  }

  scrollToBottom()
}

function disconnect() {
  if (ws) {
    ws.close()
    ws = null
  }
  connected.value = false
}

function toggleConnection() {
  if (connected.value) {
    disconnect()
  } else {
    connect()
  }
}

onMounted(() => {
  if (terminalEl.value) {
    terminalEl.value.focus()
  }
})

onUnmounted(disconnect)
</script>
