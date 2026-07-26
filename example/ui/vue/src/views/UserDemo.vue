<script setup>
import { ref } from 'vue'

const result = ref(null)
const error = ref(null)
const loading = ref(false)

async function callApi(method, path) {
  loading.value = true
  result.value = null
  error.value = null
  try {
    const res = await fetch(path, { method })
    const text = await res.text()
    let parsed
    try {
      parsed = JSON.parse(text)
    } catch {
      parsed = text
    }
    result.value = { status: res.status, body: parsed }
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <h1>API Demo</h1>
  <p>Click a button to call the corresponding endpoint.</p>

  <div class="actions">
    <button @click="callApi('GET', '/user')">GET /user</button>
    <button @click="callApi('GET', '/user/profile')">GET /user/profile</button>
    <button @click="callApi('GET', '/user/42')">GET /user/42</button>
    <button @click="callApi('GET', '/user/abc')">GET /user/abc</button>
  </div>

  <p v-if="loading" class="loading">Loading...</p>

  <div v-else-if="error" class="error">Error: {{ error }}</div>

  <div v-else-if="result" class="result">
    <h3>Response ({{ result.status }})</h3>
    <pre>{{ typeof result.body === 'string' ? result.body : JSON.stringify(result.body, null, 2) }}</pre>
  </div>
</template>

<style scoped>
.actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin: 16px 0;
}
button {
  padding: 8px 16px;
  font-size: 14px;
  cursor: pointer;
  border: 1px solid #888;
  border-radius: 4px;
  background: #fff;
}
button:hover {
  background: #e8e8e8;
}
.loading {
  color: #666;
}
.error {
  color: #c00;
}
.result pre {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
  max-width: 100%;
}
</style>
