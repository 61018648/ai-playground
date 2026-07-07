type MessageType = 'success' | 'error' | 'info'

export interface AppMessage {
  id: number
  type: MessageType
  title: string
  description?: string
}

export const useMessage = () => {
  const messages = useState<AppMessage[]>('app-messages', () => [])

  const show = (message: Omit<AppMessage, 'id'>) => {
    const id = Date.now() + Math.floor(Math.random() * 1000)
    messages.value.push({ id, ...message })
    window.setTimeout(() => {
      messages.value = messages.value.filter(item => item.id !== id)
    }, 2800)
  }

  return {
    messages,
    success: (title: string, description?: string) => show({ type: 'success', title, description }),
    error: (title: string, description?: string) => show({ type: 'error', title, description }),
    info: (title: string, description?: string) => show({ type: 'info', title, description })
  }
}
