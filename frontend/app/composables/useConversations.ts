import type { ApiConversation } from './useApi'

export const useConversations = () => {
  const api = useApi()
  const auth = useAuth()
  const conversations = useState<ApiConversation[]>('sidebar-conversations-list', () => [])
  const pending = useState('sidebar-conversations-pending', () => false)

  const loadConversations = async () => {
    if (!auth.token.value) {
      conversations.value = []
      return conversations.value
    }
    pending.value = true
    try {
      conversations.value = await api.get<ApiConversation[]>('/conversations?limit=8')
      return conversations.value
    } catch {
      conversations.value = []
      return conversations.value
    } finally {
      pending.value = false
    }
  }

  const upsertConversation = (conversation: ApiConversation) => {
    const current = Array.isArray(conversations.value) ? conversations.value : []
    conversations.value = [
      conversation,
      ...current.filter(item => item.id !== conversation.id)
    ].slice(0, 8)
  }

  const clearConversations = () => {
    conversations.value = []
  }

  return { conversations, pending, loadConversations, upsertConversation, clearConversations }
}
