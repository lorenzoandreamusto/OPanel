<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold text-white">System Monitoring</h2>
      <div class="flex items-center gap-3">
        <span :class="connected ? 'badge-success' : 'badge-danger'" class="badge">
          {{ connected ? 'Connected' : 'Disconnected' }}
        </span>
        <button @click="toggleConnection" class="btn-ghost text-sm">
          {{ connected ? 'Disconnect' : 'Connect' }}
        </button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- CPU -->
      <div class="card">
        <p class="text-sm text-opanel-text-muted mb-2">CPU Usage</p>
        <div class="flex items-end gap-3">
          <span class="text-3xl font-bold text-white">{{ stats.cpu.toFixed(1) }}%</span>
        </div>
        <div class="mt-3 w-full bg-opanel-bg rounded-full h-2">
          <div class="h-2 rounded-full transition-all duration-500"
               :class="stats.cpu > 80 ? 'bg-opanel-danger' : stats.cpu > 50 ? 'bg-opanel-warning' : 'bg-opanel-primary'"
               :style="{ width: Math.min(stats.cpu, 100) + '%' }"></div>
        </div>
      </div>

      <!-- Memory -->
      <div class="card">
        <p class="text-sm text-opanel-text-muted mb-2">Memory</p>
        <div class="flex items-end gap-3">
          <span class="text-3xl font-bold text-white">{{ stats.memory.percent.toFixed(1) }}%</span>
        </div>
        <div class="mt-3 w-full bg-opanel-bg rounded-full h-2">
          <div class="h-2 rounded-full transition-all duration-500"
               :class="stats.memory.percent > 80 ? 'bg-opanel-danger' : stats.memory.percent > 50 ? 'bg-opanel-warning' : 'bg-green-500'"
               :style="{ width: Math.min(stats.memory.percent, 100) + '%' }"></div>
        </div>
        <p class="text-xs text-opanel-text-muted mt-2">{{ formatBytes(stats.memory.used) }} / {{ formatBytes(stats.memory.total) }}</p>
      </div>

      <!-- Disk -->
      <div class="card">
        <p class="text-sm text-opanel-text-muted mb-2">Disk</p>
        <div class="flex items-end gap-3">
          <span class="text-3xl font-bold text-white">{{ stats.disk.percent.toFixed(1) }}%</span>
        </div>
        <div class="mt-3 w-full bg-opanel-bg rounded-full h-2">
          <div class="h-2 rounded-full transition-all duration-500"
               :class="stats.disk.percent > 80 ? 'bg-opanel-danger' : stats.disk.percent > 50 ? 'bg-opanel-warning' : 'bg-purple-500'"
               :style="{ width: Math.min(stats.disk.percent, 100) + '%' }"></div>
        </div>
        <p class="text-xs text-opanel-text-muted mt-2">{{ formatBytes(stats.disk.used) }} / {{ formatBytes(stats.disk.total) }}</p>
      </div>

      <!-- Load Average -->
      <div class="card">
        <p class="text-sm text-opanel-text-muted mb-2">Load Average</p>
        <div class="space-y-2">
          <div class="flex justify-between text-sm">
            <span class="text-opanel-text-muted">1 min</span>
            <span class="text-white font-medium">{{ stats.load_avg.load1.toFixed(2) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-opanel-text-muted">5 min</span>
            <span class="text-white font-medium">{{ stats.load_avg.load5.toFixed(2) }}</span>
          </div>
          <div class="flex justify-between text-sm">
            <span class="text-opanel-text-muted">15 min</span>
            <span class="text-white font-medium">{{ stats.load_avg.load15.toFixed(2) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- CPU History Chart -->
    <div class="card">
      <h3 class="text-lg font-semibold text-white mb-4">CPU History</h3>
      <div class="flex items-end gap-1 h-32">
        <div v-for="(val, i) in cpuHistory" :key="i"
             class="flex-1 rounded-t transition-all duration-300"
             :class="val > 80 ? 'bg-opanel-danger' : val > 50 ? 'bg-opanel-warning' : 'bg-opanel-primary'"
             :style="{ height: Math.max(val, 2) + '%' }"
             :title="val.toFixed(1) + '%'">
        </div>
      </div>
      <div class="flex justify-between text-xs text-opanel-text-muted mt-2">
        <span>-60s</span>
        <span>Now</span>
      </div>
    </div>

    <!-- Memory History Chart -->
    <div class="card">
      <h3 class="text-lg font-semibold text-white mb-4">Memory History</h3>
      <div class="flex items-end gap-1 h-32">
        <div v-for="(val, i) in memHistory" :key="i"
             class="flex-1 rounded-t transition-all duration-300"
             :class="val > 80 ? 'bg-opanel-danger' : val > 50 ? 'bg-opanel-warning' : 'bg-green-500'"
             :style="{ height: Math.max(val, 2) + '%' }"
             :title="val.toFixed(1) + '%'">
        </div>
      </div>
      <div class="flex justify-between text-xs text-opanel-text-muted mt-2">
        <span>-60s</span>
        <span>Now</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const stats = ref({
  cpu: 0,
  memory: { total: 0, used: 0, free: 0, available: 0, percent: 0 },
  disk: { total: 0, used: 0, free: 0, percent: 0 },
  load_avg: { load1: 0, load5: 0, load15: 0 },
  timestamp: 0,
})

const connected = ref(false)
let ws: WebSocket | null = null
const cpuHistory = ref<number[]>(new Array(30).fill(0))
const memHistory = ref<number[]>(new Array(30).fill(0))

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

function connect() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('opanel_token')
  ws = new WebSocket(`${protocol}//${window.location.host}/api/monitoring/ws?token=${token}`)

  ws.onopen = () => { connected.value = true }
  ws.onclose = () => { connected.value = false }
  ws.onerror = () => { connected.value = false }
  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      stats.value = data

      cpuHistory.value.push(data.cpu)
      if (cpuHistory.value.length > 30) cpuHistory.value.shift()

      memHistory.value.push(data.memory.percent)
      if (memHistory.value.length > 30) memHistory.value.shift()
    } catch {}
  }
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

onMounted(connect)
onUnmounted(disconnect)
</script>
