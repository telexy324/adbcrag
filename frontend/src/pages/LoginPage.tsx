import { FormEvent, useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { LogIn } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'

export function LoginPage() {
  const auth = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  if (auth.user) return <Navigate to="/" replace />

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError('')
    try {
      await auth.login(username, password)
      const from = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname || '/'
      navigate(from, { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-100 p-6">
      <Card className="w-full max-w-sm p-6">
        <h1 className="text-lg font-semibold">登录运维知识库</h1>
        <form className="mt-5 grid gap-4" onSubmit={submit}>
          <Input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名" required />
          <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="密码" required />
          {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
          <Button disabled={loading}>
            <LogIn className="h-4 w-4" />
            {loading ? '登录中...' : '登录'}
          </Button>
        </form>
      </Card>
    </div>
  )
}
