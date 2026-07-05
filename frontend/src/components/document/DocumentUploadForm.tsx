import { useQuery } from '@tanstack/react-query'
import { FormEvent, useEffect, useState } from 'react'
import { uploadDocument } from '../../api/documentApi'
import { listQualityCriteria } from '../../api/qualityCriteriaApi'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { Input } from '../ui/Input'
import { Select } from '../ui/Select'
import { Textarea } from '../ui/Textarea'
import { DocumentQualityCard } from './DocumentQualityCard'

export function DocumentUploadForm() {
  const [result, setResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [selectedCriteriaID, setSelectedCriteriaID] = useState('')
  const [criteriaContent, setCriteriaContent] = useState('')
  const { data: criteriaItems = [] } = useQuery({ queryKey: ['quality-criteria'], queryFn: listQualityCriteria })

  useEffect(() => {
    if (criteriaContent || criteriaItems.length === 0) return
    const defaultCriteria = criteriaItems.find((item) => item.isDefault) || criteriaItems[0]
    setSelectedCriteriaID(String(defaultCriteria.id))
    setCriteriaContent(defaultCriteria.content)
  }, [criteriaContent, criteriaItems])

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError('')
    const form = event.currentTarget
    try {
      const data = await uploadDocument(new FormData(form))
      setResult(data)
      form.reset()
      const selectedCriteria = criteriaItems.find((item) => String(item.id) === selectedCriteriaID)
      setCriteriaContent(selectedCriteria?.content || '')
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传失败')
    } finally {
      setLoading(false)
    }
  }

  function selectCriteria(id: string) {
    setSelectedCriteriaID(id)
    const selectedCriteria = criteriaItems.find((item) => String(item.id) === id)
    setCriteriaContent(selectedCriteria?.content || '')
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_360px]">
      <Card className="p-5">
        <form className="grid gap-4" onSubmit={onSubmit}>
          <div className="grid gap-2">
            <label className="text-sm font-medium">文档文件</label>
            <Input name="file" type="file" accept=".md,.txt,.doc,.docx,.xls,.xlsx" required />
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
          <div className="grid gap-2">
            <label className="text-sm font-medium">自定义评分标准</label>
            <Select value={selectedCriteriaID} onChange={(e) => selectCriteria(e.target.value)}>
              <option value="">临时标准</option>
              {criteriaItems.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.isDefault ? '默认 · ' : ''}
                  {item.name}
                </option>
              ))}
            </Select>
            <Textarea
              name="qualityCriteria"
              value={criteriaContent}
              onChange={(e) => {
                setCriteriaContent(e.target.value)
                setSelectedCriteriaID('')
              }}
              placeholder="例如：步骤完整性 30 分、回滚方案 20 分、风险与影响 20 分、验证方式 20 分、联系人与审批记录 10 分"
            />
          </div>
          {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
          <Button disabled={loading}>{loading ? '入库中...' : '提交入库'}</Button>
        </form>
      </Card>
      <DocumentQualityCard result={result?.qualityResult} />
    </div>
  )
}
