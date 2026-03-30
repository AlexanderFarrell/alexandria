import { ref, onMounted, onUnmounted } from 'vue'

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[]
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
  prompt(): Promise<void>
}

export function usePwaInstall() {
  const installEvent = ref<BeforeInstallPromptEvent | null>(null)
  const isInstallable = ref(false)
  const isInstalled = ref(false)

  function onBeforeInstallPrompt(e: Event) {
    e.preventDefault()
    installEvent.value = e as BeforeInstallPromptEvent
    isInstallable.value = true
  }

  function onAppInstalled() {
    installEvent.value = null
    isInstallable.value = false
    isInstalled.value = true
  }

  onMounted(() => {
    if (window.matchMedia('(display-mode: standalone)').matches) {
      isInstalled.value = true
      return
    }
    if ((navigator as Navigator & { standalone?: boolean }).standalone === true) {
      isInstalled.value = true
      return
    }
    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt)
    window.addEventListener('appinstalled', onAppInstalled)
  })

  onUnmounted(() => {
    window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt)
    window.removeEventListener('appinstalled', onAppInstalled)
  })

  async function promptInstall() {
    if (!installEvent.value) return
    await installEvent.value.prompt()
    const choice = await installEvent.value.userChoice
    if (choice.outcome === 'accepted') {
      installEvent.value = null
      isInstallable.value = false
    }
  }

  return { isInstallable, isInstalled, promptInstall }
}
