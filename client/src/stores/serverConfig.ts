import { defineStore } from 'pinia'
import { ref } from 'vue'
import { storageAdapter } from '@/utils/storage'

const SERVER_URL_KEY = 'server_url'

export const useServerConfigStore = defineStore('serverConfig', () => {
  const serverUrl = ref<string>('')

  async function load(): Promise<void> {
    serverUrl.value = (await storageAdapter.get(SERVER_URL_KEY)) ?? ''
  }

  async function save(url: string): Promise<void> {
    serverUrl.value = url.replace(/\/$/, '')
    await storageAdapter.set(SERVER_URL_KEY, serverUrl.value)
  }

  return { serverUrl, load, save }
})
