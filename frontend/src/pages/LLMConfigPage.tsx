import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { Check, Pencil, PlugZap, Save, Star, Trash2 } from 'lucide-react'
import {
  createLLMConfig,
  deleteLLMConfig,
  getActiveLLMConfig,
  listLLMConfigs,
  LLMConfig,
  LLMProvider,
  setDefaultLLMConfig,
  testLLMConfig,
  updateLLMConfig,
} from '../api/llmConfigApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'

type FormState = {
  id?: number
  name: string
  provider: LLMProvider
  baseUrl: string
  model: string
  apiKey: string
  apiSecret: string
  temperature: number
  isDefault: boolean
  enabled: boolean
}

const presets: Record<LLMProvider, Pick<FormState, 'baseUrl' | 'model'>> = {
 deepseek: { baseUrl: 'http://deepseek-v4.internal.local/v1', model: 'deepseek-v4' },
  qwen3: { baseUrl: 'http://<IP>:<Port>/Qwen3-32B/v1', model: 'Qwen3-32B' },
  openai_compatible: { baseUrl: '', model: '' },
}

const emptyForm: FormState = {
  name: '',
  provider: 'qwen3',
  baseUrl: presets.qwen3.baseUrl,
  model: presets.qwen3.model,
  apiKey: '',
  apiSecret: '',
  temperature: 0.2,
  isDefault: false,
  enabled: true,
}

export function LLMConfigPage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [message, setMessage] = useState('')
  const { data = [] } = useQuery({ queryKey: ['llm-configs'], queryFn: listLLMConfigs })
  const { data: activeConfig } = useQuery({ queryKey: ['llm-configs', 'default'], queryFn: getActiveLLMConfig })

  const saveMutation = useMutation({
    mutationFn: (value: FormState) => {
      const payload = {
        name: value.name,
        provider: value.provider,
        baseUrl: value.baseUrl,
        model: value.model,
        apiKey: value.apiKey || undefined,
        apiSecret: value.apiSecret || undefined,
        temperature: value.temperature,
        isDefault: value.isDefault,
        enabled: value.enabled,
      }
      return value.id ? updateLLMConfig(value.id, payload) : createLLMConfig(payload)
    },
    onSuccess: () => {
      setForm(emptyForm)
      setMessage('已保存')
      queryClient.invalidateQueries({ queryKey: ['llm-configs'] })
      queryClient.invalidateQueries({ queryKey: ['llm-configs', 'default'] })
    },
    onError: (err) => setMessage(err instanceof Error ? err.message : '保存失败'),
  })
  const defaultMutation = useMutation({
    mutationFn: setDefaultLLMConfig,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['llm-configs'] })
      queryClient.invalidateQueries({ queryKey: ['llm-configs', 'default'] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: deleteLLMConfig,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['llm-configs'] }),
  })
  const testMutation = useMutation({
    mutationFn: (id: number) => testLLMConfig(id),
    onSuccess: (result) => setMessage(result.ok ? `连接成功：${result.content || result.message}` : result.message),
    onError: (err) => setMessage(err instanceof Error ? err.message : '测试失败'),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(form)
  }

  function chooseProvider(provider: LLMProvider) {
    setForm({ ...form, provider, ...presets[provider] })
  }

  function edit(item: LLMConfig) {
    setForm({
      id: item.id,
      name: item.name,
      provider: item.provider,
      baseUrl: item.baseUrl,
      model: item.model,
      apiKey: '',
      apiSecret: '',
      temperature: item.temperature,
      isDefault: item.isDefault,
      enabled: item.enabled,
    })
    setMessage('')
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">模型接口</h1>
      {activeConfig && (
        <Card className="p-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div className="text-sm font-semibold">当前实际默认模型：{providerName(activeConfig.provider)} · {activeConfig.model}</div>
              <div className="mt-1 text-xs text-slate-500">{activeConfig.name} · {activeConfig.baseUrl}</div>
            </div>
            <div className={`rounded-md px-2 py-1 text-xs ${activeConfig.usingFallback ? 'bg-amber-50 text-amber-800' : activeConfig.enabled ? 'bg-teal-50 text-teal-800' : 'bg-red-50 text-red-700'}`}>
              {activeConfig.message}
            </div>
          </div>
        </Card>
      )}
      <div className="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-semibold">{form.id ? '编辑模型接口' : '新增模型接口'}</h2>
              {form.id && <Button type="button" className="bg-slate-700 hover:bg-slate-800" onClick={() => setForm(emptyForm)}>新增</Button>}
            </div>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="名称，如 Qwen3 生产接口" required />
            <Select value={form.provider} onChange={(e) => chooseProvider(e.target.value as LLMProvider)}>
              <option value="qwen3">Qwen3</option>
              <option value="deepseek">DeepSeek</option>
              <option value="openai_compatible">OpenAI Compatible</option>
            </Select>
            <Input value={form.baseUrl} onChange={(e) => setForm({ ...form, baseUrl: e.target.value })} placeholder="Base URL，例如 http://IP:Port/Qwen3-32B/v1" required />
            <Input value={form.model} onChange={(e) => setForm({ ...form, model: e.target.value })} placeholder="模型名，如 qwen3-plus" required />
            <Input type="password" value={form.apiKey} onChange={(e) => setForm({ ...form, apiKey: e.target.value })} placeholder={form.id ? 'API Key / app_key 不填则保持不变' : 'API Key / app_key'} />
            <Input type="password" value={form.apiSecret} onChange={(e) => setForm({ ...form, apiSecret: e.target.value })} placeholder={form.id ? 'API Secret / app_secret 不填则保持不变' : 'API Secret / app_secret'} />
            <Input type="number" step="0.1" min="0" max="2" value={form.temperature} onChange={(e) => setForm({ ...form, temperature: Number(e.target.value) })} placeholder="Temperature" />
            <div className="flex flex-wrap gap-4 text-sm text-slate-700">
              <label className="inline-flex items-center gap-2">
                <input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />
                启用
              </label>
              <label className="inline-flex items-center gap-2">
                <input type="checkbox" checked={form.isDefault} onChange={(e) => setForm({ ...form, isDefault: e.target.checked })} />
                设为默认
              </label>
            </div>
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
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h2 className="font-medium">{item.name}</h2>
                    {item.isDefault && <span className="inline-flex items-center gap-1 rounded-md bg-teal-50 px-2 py-1 text-xs text-teal-800"><Check className="h-3 w-3" />默认</span>}
                  </div>
                  <div className="mt-1 text-sm text-slate-500">{providerName(item.provider)} · {item.model}</div>
                  <div className="mt-1 break-all text-xs text-slate-500">{item.baseUrl}</div>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button className="px-3" onClick={() => testMutation.mutate(item.id)} title="测试连接"><PlugZap className="h-4 w-4" /></Button>
                  <Button className="px-3" onClick={() => edit(item)} title="编辑"><Pencil className="h-4 w-4" /></Button>
                  <Button className="bg-amber-700 px-3 hover:bg-amber-800 disabled:bg-slate-300" disabled={item.isDefault} onClick={() => defaultMutation.mutate(item.id)} title="设为默认"><Star className="h-4 w-4" /></Button>
                  <Button className="bg-red-700 px-3 hover:bg-red-800" onClick={() => window.confirm(`删除模型接口「${item.name}」？`) && deleteMutation.mutate(item.id)} title="删除"><Trash2 className="h-4 w-4" /></Button>
                </div>
              </div>
            </Card>
          ))}
          {data.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无模型接口配置；未配置时系统会继续使用环境变量中的 DeepSeek 接口。</Card>}
        </div>
      </div>
    </div>
  )
}

function providerName(provider: LLMProvider) {
  if (provider === 'qwen3') return 'Qwen3'
  if (provider === 'deepseek') return 'DeepSeek'
  return 'OpenAI Compatible'
}
