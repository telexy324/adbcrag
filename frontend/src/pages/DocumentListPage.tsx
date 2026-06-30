import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { listDocuments } from '../api/documentApi'
import { DocumentTable } from '../components/document/DocumentTable'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'

export function DocumentListPage() {
  const [filters, setFilters] = useState({ status: '', systemName: '', componentName: '', docType: '' })
  const { data } = useQuery({ queryKey: ['documents', filters], queryFn: () => listDocuments({ page: 1, pageSize: 20, ...filters }) })
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">文档库</h1>
      <div className="grid gap-3 md:grid-cols-4">
        <Select value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}>
          <option value="">全部状态</option>
          <option value="draft">草稿</option>
          <option value="reviewing">待审核</option>
          <option value="published">已发布</option>
          <option value="rejected">已驳回</option>
        </Select>
        <Input placeholder="系统" value={filters.systemName} onChange={(e) => setFilters({ ...filters, systemName: e.target.value })} />
        <Input placeholder="组件" value={filters.componentName} onChange={(e) => setFilters({ ...filters, componentName: e.target.value })} />
        <Input placeholder="文档类型" value={filters.docType} onChange={(e) => setFilters({ ...filters, docType: e.target.value })} />
      </div>
      <DocumentTable items={data?.items || []} />
    </div>
  )
}
