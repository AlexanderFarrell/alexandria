import { onMounted, onUnmounted } from 'vue'

export interface HotkeyDef {
  key: string
  ctrl?: boolean
  shift?: boolean
  handler: (e: KeyboardEvent) => void
  /** When true, fires even if an input/textarea has focus */
  allowInInput?: boolean
}

function isInputFocused(): boolean {
  const el = document.activeElement
  if (!el) return false
  const tag = (el as HTMLElement).tagName
  return (
    tag === 'INPUT' ||
    tag === 'TEXTAREA' ||
    tag === 'SELECT' ||
    (el as HTMLElement).isContentEditable
  )
}

export function useHotkeys(hotkeys: HotkeyDef[]) {
  function handleKeydown(e: KeyboardEvent) {
    for (const def of hotkeys) {
      const keyMatch = e.key.toLowerCase() === def.key.toLowerCase()
      // For ctrl shortcuts, match either ctrlKey or metaKey (Mac Cmd)
      const ctrlMatch = def.ctrl ? e.ctrlKey || e.metaKey : !e.ctrlKey && !e.metaKey
      const shiftMatch = def.shift ? e.shiftKey : !e.shiftKey

      if (keyMatch && ctrlMatch && shiftMatch) {
        if (!def.allowInInput && isInputFocused()) continue
        e.preventDefault()
        def.handler(e)
        return
      }
    }
  }

  onMounted(() => window.addEventListener('keydown', handleKeydown))
  onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
}
