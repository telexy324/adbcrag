import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { CheckCircle2, Pencil, PlugZap, Save, Trash2 } from 'lucide-react'
import { createK8sCluster, deleteK8sCluster, K8sCluster, K8sClusterInput, listK8sClusters, testK8sCluster, updateK8sCluster } from '../api/k8sApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Textarea } from '../components/ui/Textarea'

type FormState = K8sClusterInput & { id?: number; namespacesText: string }

const emptyForm: FormState = {
  name: '',
  clusterCode: '',
  apiServer: '',
  authType: 'bearer_token',
  bearerToken: '',
  caCert: '',
  namespacesText: '',
  insecureSkipTLSVerify: false,
  enabled: true,
}

export function K8sClusterPage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [message, setMessage] = useState('')
  const { data = [] } = useQuery({ queryKey: ['k8s-clusters'], queryFn: listK8sClusters })

  const saveMutation = useMutation({
    mutationFn: (value: FormState) => {
      const payload = normalizeForm(value)
      return value.id ? updateK8sCluster(value.id, payload) : createK8sCluster(payload)
    },
    onSuccess: () => {
      setForm(emptyForm)
      setMessage('已保存')
      queryClient.invalidateQueries({ queryKey: ['k8s-clusters'] })
    },
    onError: (err) => setMessage(err instanceof Error ? err.message : '保存失败'),
  })
  const deleteMutation = useMutation({
    mutationFn: deleteK8sCluster,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['k8s-clusters'] }),
  })
  const testMutation = useMutation({
    mutationFn: testK8sCluster,
    onSuccess: (result) => setMessage(result.message),
    onError: (err) => setMessage(err instanceof Error ? err.message : '连接测试失败'),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(form)
  }

  function edit(item: K8sCluster) {
    setForm({
      ...item,
      bearerToken: '',
      caCert: '',
      allowedNamespaces: item.allowedNamespaces || [],
      namespacesText: (item.allowedNamespaces || []).join('\n'),
    })
    setMessage('')
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">K8s 集群</h1>
      <div className="grid gap-5 xl:grid-cols-[460px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold">{form.id ? '编辑集群' : '新增集群'}</h2>
              {form.id && <Button type="button" className="bg-slate-700 hover:bg-slate-800" onClick={() => setForm(emptyForm)}>新增</Button>}
            </div>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="集群名称，如 生产 K8s" required />
            <Input value={form.clusterCode} onChange={(e) => setForm({ ...form, clusterCode: e.target.value })} placeholder="集群标识，如 prod-k8s-01" required />
            <Input value={form.apiServer} onChange={(e) => setForm({ ...form, apiServer: e.target.value })} placeholder="API Server，如 https://10.0.0.1:6443" required />
            <Select value={form.authType || 'bearer_token'} onChange={(e) => setForm({ ...form, authType: e.target.value as 'bearer_token' })}>
              <option value="bearer_token">Bearer Token</option>
            </Select>
            <Textarea value={form.bearerToken || ''} onChange={(e) => setForm({ ...form, bearerToken: e.target.value })} placeholder="ServiceAccount Token，编辑时留空表示不更新" />
            <Textarea value={form.caCert || ''} onChange={(e) => setForm({ ...form, caCert: e.target.value })} placeholder="CA 证书 PEM，可选" />
            <Textarea value={form.namespacesText} onChange={(e) => setForm({ ...form, namespacesText: e.target.value })} placeholder="允许读取的 namespace，每行一个；留空表示不限制" />
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input type="checkbox" checked={!!form.insecureSkipTLSVerify} onChange={(e) => setForm({ ...form, insecureSkipTLSVerify: e.target.checked })} />
              跳过 TLS 证书校验
            </label>
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
                  <div className="mt-1 text-sm text-slate-500">{item.clusterCode} · {item.apiServer}</div>
                  <div className="mt-1 text-xs text-slate-500">Namespaces: {(item.allowedNamespaces || []).join(', ') || '不限制'}</div>
                </div>
                <div className="flex gap-2">
                  <Button className="px-3" onClick={() => testMutation.mutate(item.id)} title="测试连接"><PlugZap className="h-4 w-4" /></Button>
                  <Button className="px-3" onClick={() => edit(item)} title="编辑"><Pencil className="h-4 w-4" /></Button>
                  <Button className="bg-red-700 px-3 hover:bg-red-800" onClick={() => window.confirm(`删除集群「${item.name}」？`) && deleteMutation.mutate(item.id)} title="删除"><Trash2 className="h-4 w-4" /></Button>
                </div>
              </div>
            </Card>
          ))}
          {data.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无 K8s 集群</Card>}
        </div>
      </div>
    </div>
  )
}

function normalizeForm(form: FormState): K8sClusterInput {
  return {
    ...form,
    allowedNamespaces: form.namespacesText.split('\n').map((item) => item.trim()).filter(Boolean),
  }
}
