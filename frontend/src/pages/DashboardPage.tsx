import { useQuery } from '@tanstack/react-query'
import { getDashboardStats } from '../api/documentApi'
import { Card } from '../components/ui/Card'

export function DashboardPage() {
  const { data } = useQuery({ queryKey: ['dashboard-stats'], queryFn: getDashboardStats })
  const cards = [
    ['文档总数', data?.documentTotal ?? 0],
    ['已发布', data?.publishedTotal ?? 0],
    ['待审核', data?.reviewingTotal ?? 0],
    ['平均质量分', Number(data?.averageQualityScore ?? 0).toFixed(1)],
  ]
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">运维知识库概览</h1>
      <div className="grid gap-4 md:grid-cols-4">
        {cards.map(([label, value]) => (
          <Card key={label} className="p-5">
            <div className="text-sm text-slate-500">{label}</div>
            <div className="mt-2 text-3xl font-semibold text-slate-900">{value}</div>
          </Card>
        ))}
      </div>
    </div>
  )
}
