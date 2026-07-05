import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChangeEvent, FormEvent, useState } from 'react'
import { Check, FileUp, Pencil, Save, Star, Trash2 } from 'lucide-react'
import {
  createQualityCriteria,
  deleteQualityCriteria,
  listQualityCriteria,
  QualityCriteria,
  setDefaultQualityCriteria,
  updateQualityCriteria,
} from '../api/qualityCriteriaApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Textarea } from '../components/ui/Textarea'

type FormState = {
  id?: number
  name: string
  content: string
  isDefault: boolean
}

const emptyForm: FormState = {
  name: '',
  content: '',
  isDefault: false,
}

export function QualityCriteriaPage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<FormState>(emptyForm)
  const [error, setError] = useState('')
  const { data = [] } = useQuery({ queryKey: ['quality-criteria'], queryFn: listQualityCriteria })

  const saveMutation = useMutation({
    mutationFn: (value: FormState) =>
      value.id
        ? updateQualityCriteria(value.id, { name: value.name, content: value.content, isDefault: value.isDefault })
        : createQualityCriteria({ name: value.name, content: value.content, isDefault: value.isDefault }),
    onSuccess: () => {
      setForm(emptyForm)
      setError('')
      queryClient.invalidateQueries({ queryKey: ['quality-criteria'] })
    },
    onError: (err) => setError(err instanceof Error ? err.message : '保存失败'),
  })

  const defaultMutation = useMutation({
    mutationFn: setDefaultQualityCriteria,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['quality-criteria'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteQualityCriteria,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['quality-criteria'] }),
  })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    saveMutation.mutate(form)
  }

  async function importFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return
    const content = await file.text()
    setForm((value) => ({
      ...value,
      name: value.name || file.name.replace(/\.(txt|md)$/i, ''),
      content,
    }))
    event.target.value = ''
  }

  function editCriteria(item: QualityCriteria) {
    setForm({ id: item.id, name: item.name, content: item.content, isDefault: item.isDefault })
    setError('')
  }

  function removeCriteria(item: QualityCriteria) {
    if (window.confirm(`删除评分标准「${item.name}」？`)) {
      deleteMutation.mutate(item.id)
    }
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">评分标准</h1>
      <div className="grid gap-5 xl:grid-cols-[420px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-semibold">{form.id ? '编辑标准' : '新增标准'}</h2>
              <label className="inline-flex h-9 cursor-pointer items-center gap-2 rounded-md border border-border px-3 text-sm text-slate-700 hover:bg-muted">
                <FileUp className="h-4 w-4" />
                导入
                <input className="hidden" type="file" accept=".txt,.md,text/plain,text/markdown" onChange={importFile} />
              </label>
            </div>
            <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="标准名称" required />
            <Textarea
              className="min-h-72"
              value={form.content}
              onChange={(e) => setForm({ ...form, content: e.target.value })}
              placeholder="步骤完整性 30 分&#10;风险与影响 20 分&#10;回滚方案 20 分&#10;验证方式 20 分&#10;联系人与审批记录 10 分"
              required
            />
            <label className="inline-flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={form.isDefault}
                onChange={(e) => setForm({ ...form, isDefault: e.target.checked })}
              />
              设为默认标准
            </label>
            {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
            <div className="flex gap-2">
              <Button disabled={saveMutation.isPending}>
                <Save className="h-4 w-4" />
                {saveMutation.isPending ? '保存中...' : '保存'}
              </Button>
              {form.id && (
                <Button type="button" className="bg-slate-700 hover:bg-slate-800" onClick={() => setForm(emptyForm)}>
                  新增
                </Button>
              )}
            </div>
          </form>
        </Card>

        <div className="space-y-3">
          {data.map((item) => (
            <Card key={item.id} className="p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <h2 className="font-medium">{item.name}</h2>
                    {item.isDefault && (
                      <span className="inline-flex items-center gap-1 rounded-md bg-teal-50 px-2 py-1 text-xs text-teal-800">
                        <Check className="h-3 w-3" />
                        默认
                      </span>
                    )}
                  </div>
                  <div className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap text-sm text-slate-600">{item.content}</div>
                </div>
                <div className="flex shrink-0 gap-2">
                  <Button type="button" className="px-3" onClick={() => editCriteria(item)} title="编辑">
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    type="button"
                    className="bg-amber-700 px-3 hover:bg-amber-800 disabled:bg-slate-300"
                    disabled={item.isDefault}
                    onClick={() => defaultMutation.mutate(item.id)}
                    title="设为默认"
                  >
                    <Star className="h-4 w-4" />
                  </Button>
                  <Button type="button" className="bg-red-700 px-3 hover:bg-red-800" onClick={() => removeCriteria(item)} title="删除">
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </Card>
          ))}
          {data.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无评分标准</Card>}
        </div>
      </div>
    </div>
  )
}
