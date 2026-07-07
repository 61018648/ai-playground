import type { ApiUser } from './useApi'

interface AuthResponse {
  user: ApiUser
  accessToken: string
}

export const useAuth = () => {
  const api = useApi()
  const token = useCookie<string | null>('image_ai_token', {
    sameSite: 'lax',
    maxAge: 60 * 60 * 24 * 7
  })
  const user = useState<ApiUser | null>('auth-user', () => null)
  const loading = useState('auth-loading', () => false)

  const setSession = (session: AuthResponse) => {
    token.value = session.accessToken
    user.value = session.user
  }

  const sendCode = (email: string, purpose: 'register' | 'forgot_password' | 'login' | 'change_email') => {
    return api.post<{ expiresIn: number, devCode?: string }>('/auth/send-code', { email, purpose })
  }

  const login = async (payload: { email: string, password: string, remember: boolean }) => {
    loading.value = true
    try {
      const session = await api.post<AuthResponse>('/auth/login', payload)
      setSession(session)
      return session
    } finally {
      loading.value = false
    }
  }

  const register = async (payload: { email: string, password: string, code: string, inviteCode?: string }) => {
    loading.value = true
    try {
      const session = await api.post<AuthResponse>('/auth/register', payload)
      setSession(session)
      return session
    } finally {
      loading.value = false
    }
  }

  const forgotPassword = async (payload: { email: string, code: string, newPassword: string }) => {
    loading.value = true
    try {
      return await api.post<{ ok: boolean }>('/auth/forgot-password', payload)
    } finally {
      loading.value = false
    }
  }

  const loadMe = async () => {
    if (!token.value || user.value) return user.value
    try {
      user.value = await api.get<ApiUser>('/auth/me')
    } catch {
      token.value = null
      user.value = null
    }
    return user.value
  }

  const logout = () => {
    token.value = null
    user.value = null
  }

  return { token, user, loading, sendCode, login, register, forgotPassword, loadMe, logout }
}
