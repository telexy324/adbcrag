import { Card } from '../ui/Card'

export function AnswerCard({ answer }: { answer?: string }) {
  if (!answer) return null
  return (
    <Card className="p-5">
      <div className="mb-3 rounded-md bg-amber-50 p-3 text-sm text-amber-900">
        AI 回答仅供运维排查参考，生产操作请遵守变更审批流程。
      </div>
      <div className="whitespace-pre-wrap text-sm leading-7 text-slate-800">{answer}</div>
    </Card>
  )
}
