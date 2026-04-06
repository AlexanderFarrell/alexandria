export interface StorageAdapter {
  get(key: string): Promise<string | null>
  set(key: string, value: string): Promise<void>
  remove(key: string): Promise<void>
}

class WebStorageAdapter implements StorageAdapter {
  async get(key: string): Promise<string | null> {
    return localStorage.getItem(key)
  }
  async set(key: string, value: string): Promise<void> {
    localStorage.setItem(key, value)
  }
  async remove(key: string): Promise<void> {
    localStorage.removeItem(key)
  }
}

class TauriStorageAdapter implements StorageAdapter {
  private store: import('@tauri-apps/plugin-store').Store | null = null

  private async getStore(): Promise<import('@tauri-apps/plugin-store').Store> {
    if (!this.store) {
      const { load } = await import('@tauri-apps/plugin-store')
      this.store = await load('alexandria.json', { defaults: {}, autoSave: true })
    }
    return this.store
  }

  async get(key: string): Promise<string | null> {
    const s = await this.getStore()
    return (await s.get<string>(key)) ?? null
  }

  async set(key: string, value: string): Promise<void> {
    const s = await this.getStore()
    await s.set(key, value)
  }

  async remove(key: string): Promise<void> {
    const s = await this.getStore()
    await s.delete(key)
  }
}

function createStorageAdapter(): StorageAdapter {
  if (typeof (window as Window & { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ !== 'undefined') {
    return new TauriStorageAdapter()
  }
  return new WebStorageAdapter()
}

export const storageAdapter = createStorageAdapter()
