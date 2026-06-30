import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { listDocuments, reviewDocument } from '../../api/documentApi'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'

export function ReviewPanel() {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['reviewing-documents'], queryFn: () => listDocuments({ page: 1, pageSize: 50, status: 'reviewing' }) })
  const mutation = useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'approve' | 'reject' | 'deprecate' }) => reviewDocument(id, { action }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['reviewing-documents'] }),
  })
  return (
    <div className="space-y-3">
      {(data?.items || []).map((doc) => (
        <Card key={doc.id} className="flex items-center justify-between p-4">
          <div>
            <div className="font-medium">{doc.title}</div>
            <div className="mt-1 text-sm text-slate-500">{doc.systemName || '-'} · {doc.componentName || '-'} · 质量分 {doc.qualityScore}</div>
          </div>
          <div className="flex gap-2">
            <Button onClick={() => mutation.mutate({ id: doc.id, action: 'approve' })}>通过</Button>
            <Button className="bg-red-700 hover:bg-red-800" onClick={() => mutation.mutate({ id: doc.id, action: 'reject' })}>驳回</Button>
            <Button className="bg-slate-700 hover:bg-slate-800" onClick={() => mutation.mutate({ id: doc.id, action: 'deprecate' })}>废弃</Button>
          </div>
        </Card>
      ))}
      {data?.items?.length === 0 && <Card className="p-8 text-center text-sm text-slate-500">暂无待审核文档</Card>}
    </div>
  )
}
