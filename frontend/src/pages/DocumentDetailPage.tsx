import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { getDocument } from '../api/documentApi'
import { DocumentQualityCard } from '../components/document/DocumentQualityCard'
import { DocumentStatusBadge } from '../components/document/DocumentStatusBadge'
import { Card } from '../components/ui/Card'

export function DocumentDetailPage() {
  const id = Number(useParams().id)
  const { data } = useQuery({ queryKey: ['document', id], queryFn: () => getDocument(id), enabled: Boolean(id) })
  if (!data) return null
  return (
    <div className="space-y-5">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-semibold">{data.title}</h1>
        <DocumentStatusBadge status={data.status} />
      </div>
      <Card className="grid gap-3 p-5 text-sm md:grid-cols-3">
        <div>系统：{data.systemName || '-'}</div>
        <div>组件：{data.componentName || '-'}</div>
        <div>类型：{data.docType || '-'}</div>
        <div className="md:col-span-3">摘要：{data.summary || '-'}</div>
      </Card>
      <DocumentQualityCard result={data.qualityResult} />
    </div>
  )
}
