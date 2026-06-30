import { FormEvent, useState } from 'react'
import { uploadDocument } from '../../api/documentApi'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { DocumentQualityCard } from './DocumentQualityCard'

export function DocumentUploadForm() {
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError('')
    const form = event.currentTarget
    try {
      const data = await uploadDocument(new FormData(form))
      setResult(data)
      form.reset()
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
      <Card className="p-5">
        <form className="grid gap-4" onSubmit={onSubmit}>
          <div className="grid gap-2">
            <label className="text-sm font-medium">文档文件</label>
            <Input name="file" type="file" accept=".md,.txt" required />
          </div>
          <div className="grid gap-2">
            <label className="text-sm font-medium">文档标题</label>
            <Input name="title" placeholder="Redis 内存告警处置手册" />
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            <Input name="systemName" placeholder="所属系统" />
            <Input name="componentName" placeholder="组件，如 Redis" />
            <Select name="docType" defaultValue="">
              <option value="">文档类型</option>
              <option value="告警处置">告警处置</option>
              <option value="应急预案">应急预案</option>
              <option value="启停手册">启停手册</option>
              <option value="变更回滚">变更回滚</option>
            </Select>
          </div>
          <Input name="tags" placeholder="标签，逗号分隔" />
          {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
          <Button disabled={loading}>{loading ? '入库中...' : '提交入库'}</Button>
        </form>
      </Card>
      <DocumentQualityCard result={result?.qualityResult} />
    </div>
  )
}
