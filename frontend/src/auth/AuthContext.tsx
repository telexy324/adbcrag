import { createContext, ReactNode, useContext, useEffect, useMemo, useState } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { CurrentUser, login as loginApi, me } from '../api/authApi'

type AuthContextValue = {
  user: CurrentUser | null
  token: string
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState(() => localStorage.getItem('accessToken') || '')
  const [user, setUser] = useState<CurrentUser | null>(() => {
    const raw = localStorage.getItem('currentUser')
    return raw ? JSON.parse(raw) as CurrentUser : null
  })
  const [loading, setLoading] = useState(!!token)

  useEffect(() => {
    if (!token) {
      setLoading(false)
      return
    }
    me().then((current) => {
      setUser(current)
      localStorage.setItem('currentUser', JSON.stringify(current))
    }).catch(() => {
      setToken('')
      setUser(null)
      localStorage.removeItem('accessToken')
      localStorage.removeItem('currentUser')
    }).finally(() => setLoading(false))
  }, [token])

  const value = useMemo<AuthContextValue>(() => ({
    user,
    token,
    loading,
    login: async (username, password) => {
      const result = await loginApi(username, password)
      localStorage.setItem('accessToken', result.accessToken)
      localStorage.setItem('currentUser', JSON.stringify(result.user))
      setToken(result.accessToken)
      setUser(result.user)
    },
    logout: () => {
      localStorage.removeItem('accessToken')
      localStorage.removeItem('currentUser')
      setToken('')
      setUser(null)
    },
  }), [user, token, loading])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const value = useContext(AuthContext)
  if (!value) throw new Error('AuthProvider is missing')
  return value
}

export function RequireAuth({ children, adminOnly = false }: { children: ReactNode; adminOnly?: boolean }) {
  const auth = useAuth()
  const location = useLocation()
  if (auth.loading) return <div className="p-6 text-sm text-slate-500">加载中...</div>
  if (!auth.user) return <Navigate to="/login" state={{ from: location }} replace />
  if (adminOnly && auth.user.role !== 'admin') return <Navigate to="/" replace />
  return <>{children}</>
}
