import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { CheckCircle2, Pencil, PlugZap, Save, Trash2 } from 'lucide-react'
import { createLogSource, deleteLogSource, listLogSources, LogSource, LogSourceInput, testLogSource, updateLogSource } from '../api/logApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Textarea } from '../components/ui/Textarea'

type FormState = LogSourceInput & { id?: number; allowlistText: string }

const emptyForm: FormState = {
  name: '',
  sourceType: 'elasticsearch',
  authType: 'password',
  serverPort: 22,
  esTimeField: '@timestamp',
  enabled: true,
  allowlistText: '',
}

export function LogSourcePage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [message, setMessage] = useState('')
  const { data = [] } = useQuery({ queryKey: ['log-sources'], queryFn: listLogSources })

  const saveMutation = useMutation({
    mutationFn: (value: FormState) => {
      const payload = normalizeForm(value)
      return value.id ? updateLogSource(value.id, payload) : createLogSource(payload)
    },
    onSuccess: () => {
      setForm(emptyForm)
      setMessage('已保存')
      queryClient.invalidateQueries({ queryKey: ['log-sources'] })
    },
    onError: (err) => setMessage(err instanceof Error ? err.message : '保存失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteLogSource,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['log-sources'] }),
  })
  const testMutation = useMutation({
    mutationFn: testLogSource,
    onSuccess: (result) => setMessage(result.message),
    onError: (err) => setMessage(err instanceof Error ? err.message : '连接测试失败'),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(form)
  }

  function edit(item: LogSource) {
    const pathAllowlist = item.pathAllowlist || []
    setForm({
      ...item,
      pathAllowlist,
      password: '',
      privateKey: '',
      privateKeyPassphrase: '',
      allowlistText: pathAllowlist.join('\n'),
    })
    setMessage('')
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">日志源</h1>
      <div className="grid gap-5 xl:grid-cols-[440px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold">{form.id ? '编辑日志源' : '新增日志源'}</h2>
              {form.id && <Button type="button" className="bg-slate-700 hover:bg-slate-800" onClick={() => setForm(emptyForm)}>新增</Button>}
            </div>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="日志源名称" required />
            <Select value={form.sourceType} onChange={(e) => setForm({ ...emptyForm, sourceType: e.target.value as FormState['sourceType'] })}>
              <option value="elasticsearch">Elasticsearch</option>
              <option value="server_file">服务器日志文件</option>
            </Select>
            <div className="grid gap-3 md:grid-cols-3">
              <Input value={form.systemName || ''} onChange={(e) => setForm({ ...form, systemName: e.target.value })} placeholder="系统" />
              <Input value={form.componentName || ''} onChange={(e) => setForm({ ...form, componentName: e.target.value })} placeholder="组件" />
              <Input value={form.environment || ''} onChange={(e) => setForm({ ...form, environment: e.target.value })} placeholder="环境" />
            </div>

            {form.sourceType === 'elasticsearch' ? (
              <>
                <Input value={form.endpoint || ''} onChange={(e) => setForm({ ...form, endpoint: e.target.value })} placeholder="ES 地址，如 https://es.local:9200" required />
                <div className="grid gap-3 md:grid-cols-2">
                  <Input value={form.username || ''} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder="账号" />
                  <Input type="password" value={form.password || ''} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="密码" />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Input value={form.esIndexPattern || ''} onChange={(e) => setForm({ ...form, esIndexPattern: e.target.value })} placeholder="索引，如 app-log-*" required />
                  <Input value={form.esTimeField || ''} onChange={(e) => setForm({ ...form, esTimeField: e.target.value })} placeholder="时间字段，如 @timestamp" />
                </div>
              </>
            ) : (
              <>
                <div className="grid gap-3 md:grid-cols-[1fr_120px]">
                  <Input value={form.serverHost || ''} onChange={(e) => setForm({ ...form, serverHost: e.target.value })} placeholder="服务器地址" required />
                  <Input type="number" value={form.serverPort || 22} onChange={(e) => setForm({ ...form, serverPort: Number(e.target.value) })} placeholder="端口" />
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <Input value={form.username || ''} onChange={(e) => setForm({ ...form, username: e.target.value })} placeholder="账号" required />
                  <Select value={form.authType || 'password'} onChange={(e) => setForm({ ...form, authType: e.target.value as FormState['authType'] })}>
                    <option value="password">账号密码</option>
                    <option value="private_key">私钥</option>
                  </Select>
                </div>
                {form.authType === 'private_key' ? (
                  <>
                    <Textarea value={form.privateKey || ''} onChange={(e) => setForm({ ...form, privateKey: e.target.value })} placeholder="OpenSSH 私钥" />
                    <Input type="password" value={form.privateKeyPassphrase || ''} onChange={(e) => setForm({ ...form, privateKeyPassphrase: e.target.value })} placeholder="私钥口令，可选" />
                  </>
                ) : (
                  <Input type="password" value={form.password || ''} onChange={(e) => setForm({ ...form, password: e.target.value })} placeholder="密码" />
                )}
                <Input value={form.logPath || ''} onChange={(e) => setForm({ ...form, logPath: e.target.value })} placeholder="日志路径，如 /data/app/logs/app.log" required />
                <Textarea value={form.allowlistText} onChange={(e) => setForm({ ...form, allowlistText: e.target.value })} placeholder="/data/app/logs/&#10;/data/app/archive/" required />
              </>
            )}
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input type="checkbox" checked={form.enabled !== false} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
              启用
            </label>
            {message && <div className="rounded-md bg-slate-50 p-3 text-sm text-slate-700">{message}</div>}
            <Button disabled={saveMutation.isPending}>
              <Save className="h-4 w-4" />
              {saveMutation.isPending ? '保存中...' : '保存'}
            </Button>
          </form>
        </Card>

        <div className="space-y-3">
          {data.map((item) => (
            <Card key={item.id} className="p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="font-medium">{item.name}</h2>
                    {item.enabled && <CheckCircle2 className="h-4 w-4 text-teal-700" />}
                  </div>
                  <div className="mt-1 text-sm text-slate-500">{item.sourceType === 'elasticsearch' ? item.endpoint : `${item.serverHost}:${item.serverPort}${item.logPath}`}</div>
                  <div className="mt-1 text-xs text-slate-500">{item.systemName || '-'} · {item.componentName || '-'} · {item.environment || '-'}</div>
                </div>
                <div className="flex gap-2">
                  <Button className="px-3" onClick={() => testMutation.mutate(item.id)} title="测试连接"><PlugZap className="h-4 w-4" /></Button>
                  <Button className="px-3" onClick={() => edit(item)} title="编辑"><Pencil className="h-4 w-4" /></Button>
                  <Button className="bg-red-700 px-3 hover:bg-red-800" onClick={() => window.confirm(`删除日志源「${item.name}」？`) && deleteMutation.mutate(item.id)} title="删除"><Trash2 className="h-4 w-4" /></Button>
                </div>
              </div>
            </Card>
          ))}
          {data.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无日志源</Card>}
        </div>
      </div>
    </div>
  )
}

function normalizeForm(form: FormState): LogSourceInput {
  return {
    ...form,
    pathAllowlist: form.allowlistText.split('\n').map((item) => item.trim()).filter(Boolean),
  }
}
