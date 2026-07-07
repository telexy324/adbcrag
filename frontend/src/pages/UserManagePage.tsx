import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { KeyRound, Save } from 'lucide-react'
import { createUser, listUsers, resetUserPassword, updateUser } from '../api/authApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'

type FormState = {
  username: string
  displayName: string
  password: string
  role: 'admin' | 'user'
  enabled: boolean
}

const emptyForm: FormState = { username: '', displayName: '', password: '', role: 'user', enabled: true }

export function UserManagePage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [message, setMessage] = useState('')
  const { data = [] } = useQuery({ queryKey: ['users'], queryFn: listUsers })

  const createMutation = useMutation({
    mutationFn: createUser,
    onSuccess: () => {
      setForm(emptyForm)
      setMessage('用户已创建')
      queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    onError: (err) => setMessage(err instanceof Error ? err.message : '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, enabled, role, displayName }: { id: number; enabled?: boolean; role?: 'admin' | 'user'; displayName?: string }) => updateUser(id, { enabled, role, displayName }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['users'] }),
  })

  const resetMutation = useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) => resetUserPassword(id, password),
    onSuccess: () => setMessage('密码已重置'),
    onError: (err) => setMessage(err instanceof Error ? err.message : '重置失败'),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    createMutation.mutate(form)
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">用户管理</h1>
      <div className="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <h2 className="text-sm font-semibold">新增用户</h2>
            <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder="登录账号" required />
            <Input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} placeholder="显示名称" />
            <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="初始密码，至少 8 位" required />
            <Select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value as FormState['role'] })}>
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
            </Select>
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
              启用
            </label>
            {message && <div className="rounded-md bg-slate-50 p-3 text-sm text-slate-700">{message}</div>}
            <Button disabled={createMutation.isPending}>
              <Save className="h-4 w-4" />
              {createMutation.isPending ? '保存中...' : '创建用户'}
            </Button>
          </form>
        </Card>

        <div className="space-y-3">
          {data.map((user) => (
            <Card key={user.id} className="p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div className="font-medium">{user.displayName || user.username}</div>
                  <div className="mt-1 text-sm text-slate-500">{user.username} · {user.role} · {user.enabled ? '启用' : '禁用'}</div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Select className="h-9 w-28" value={user.role} onChange={(e) => updateMutation.mutate({ id: user.id, role: e.target.value as 'admin' | 'user' })}>
                    <option value="user">普通用户</option>
                    <option value="admin">管理员</option>
                  </Select>
                  <Button className="bg-slate-700 px-3 hover:bg-slate-800" onClick={() => updateMutation.mutate({ id: user.id, enabled: !user.enabled })}>
                    {user.enabled ? '禁用' : '启用'}
                  </Button>
                  <Button className="px-3" onClick={() => {
                    const password = window.prompt(`重置 ${user.username} 的密码`)
                    if (password) resetMutation.mutate({ id: user.id, password })
                  }}>
                    <KeyRound className="h-4 w-4" />
                    重置密码
                  </Button>
                </div>
              </div>
            </Card>
          ))}
          {data.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无用户</Card>}
        </div>
      </div>
    </div>
  )
}
