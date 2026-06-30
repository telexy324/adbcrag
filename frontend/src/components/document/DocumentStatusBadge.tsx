import { Badge } from '../ui/Badge'

const labels: Record<string, string> = {
  draft: '草稿',
  reviewing: '待审核',
  published: '已发布',
  archived: '已归档',
  deprecated: '已废弃',
  rejected: '已驳回',
}

const colors: Record<string, string> = {
  draft: 'bg-slate-100 text-slate-700',
  reviewing: 'bg-amber-100 text-amber-800',
  published: 'bg-emerald-100 text-emerald-800',
  archived: 'bg-slate-200 text-slate-700',
  deprecated: 'bg-rose-100 text-rose-800',
  rejected: 'bg-red-100 text-red-800',
}

export function DocumentStatusBadge({ status }: { status: string }) {
  return <Badge className={colors[status] || colors.draft}>{labels[status] || status}</Badge>
}
