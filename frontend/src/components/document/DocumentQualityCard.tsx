import { Card } from '../ui/Card'

export function DocumentQualityCard({ result }: { result: any }) {
  if (!result) return null
  return (
    <Card className="p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold">AI 质量检查</h3>
        <span className="text-2xl font-semibold text-teal-700">{result.score ?? '-'}</span>
      </div>
      <p className="mt-2 text-sm text-slate-600">{result.summary}</p>
      {result.criteria && (
        <div className="mt-3 rounded-md bg-slate-50 p-3 text-sm text-slate-600">
          <div className="mb-1 font-medium text-slate-700">评分标准</div>
          <div className="max-h-32 overflow-auto whitespace-pre-wrap">{result.criteria}</div>
        </div>
      )}
      <div className="mt-3 space-y-2">
        {(result.problems || []).map((item: any, index: number) => (
          <div key={index} className="rounded-md bg-amber-50 p-3 text-sm text-amber-900">
            <div className="font-medium">{item.type || '问题'}</div>
            <div>{item.description}</div>
            {item.suggestion && <div className="mt-1 text-amber-800">建议：{item.suggestion}</div>}
          </div>
        ))}
      </div>
    </Card>
  )
}
