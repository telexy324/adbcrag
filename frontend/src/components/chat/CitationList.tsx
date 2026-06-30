import { Card } from '../ui/Card'

export function CitationList({ citations }: { citations: any[] }) {
  if (!citations?.length) return null
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold">引用来源</h3>
      {citations.map((item) => (
        <Card key={item.chunkId} className="p-4">
          <div className="flex items-center justify-between text-sm">
            <span className="font-medium">《{item.documentTitle}》{item.sourceSection ? ` · ${item.sourceSection}` : ''}</span>
            <span className="text-xs text-slate-500">score {Number(item.score || 0).toFixed(3)}</span>
          </div>
          <p className="mt-2 line-clamp-4 text-sm leading-6 text-slate-600">{item.content}</p>
        </Card>
      ))}
    </div>
  )
}
