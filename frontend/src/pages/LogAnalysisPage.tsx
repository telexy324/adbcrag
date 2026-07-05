import { useMutation, useQuery } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { Search, Sparkles } from 'lucide-react'
import { analyzeLogs, listLogSources, LogItem, previewLogs } from '../api/logApi'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Textarea } from '../components/ui/Textarea'
import { CitationList } from '../components/chat/CitationList'

type QueryState = {
  sourceId: string
  timeStart: string
  timeEnd: string
  keyword: string
  logLevel: string
  logPath: string
  question: string
}

const emptyQuery: QueryState = {
  sourceId: '',
  timeStart: '',
  timeEnd: '',
  keyword: '',
  logLevel: '',
  logPath: '',
  question: '',
}

export function LogAnalysisPage() {
  const [query, setQuery] = useState<QueryState>(emptyQuery)
  const { data: sources = [] } = useQuery({ queryKey: ['log-sources'], queryFn: listLogSources })
  const previewMutation = useMutation({ mutationFn: previewLogs })
  const analysisMutation = useMutation({ mutationFn: analyzeLogs })

  const selectedSource = sources.find((item) => String(item.id) === query.sourceId)

  function payload() {
    return {
      sourceId: Number(query.sourceId),
      timeStart: query.timeStart,
      timeEnd: query.timeEnd,
      keyword: query.keyword,
      logLevel: query.logLevel,
      logPath: query.logPath,
      question: query.question,
      systemName: selectedSource?.systemName,
      componentName: selectedSource?.componentName,
      limit: 100,
    }
  }

  function preview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    previewMutation.mutate(payload())
  }

  function analyze() {
    analysisMutation.mutate(payload())
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">日志分析</h1>
      <div className="grid gap-5 xl:grid-cols-[440px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={preview}>
            <Select value={query.sourceId} onChange={(e) => setQuery({ ...query, sourceId: e.target.value })} required>
              <option value="">选择日志源</option>
              {sources.map((source) => (
                <option key={source.id} value={source.id}>{source.name}</option>
              ))}
            </Select>
            <div className="grid gap-3 md:grid-cols-2">
              <Input type="datetime-local" value={query.timeStart} onChange={(e) => setQuery({ ...query, timeStart: e.target.value })} />
              <Input type="datetime-local" value={query.timeEnd} onChange={(e) => setQuery({ ...query, timeEnd: e.target.value })} />
            </div>
            <div className="grid gap-3 md:grid-cols-2">
              <Input value={query.keyword} onChange={(e) => setQuery({ ...query, keyword: e.target.value })} placeholder="关键词，如 ERROR timeout" />
              <Select value={query.logLevel} onChange={(e) => setQuery({ ...query, logLevel: e.target.value })}>
                <option value="">全部级别</option>
                <option value="ERROR">ERROR</option>
                <option value="WARN">WARN</option>
                <option value="INFO">INFO</option>
              </Select>
            </div>
            {selectedSource?.sourceType === 'server_file' && (
              <Input value={query.logPath} onChange={(e) => setQuery({ ...query, logPath: e.target.value })} placeholder={selectedSource.logPath || '日志路径'} />
            )}
            <Textarea value={query.question} onChange={(e) => setQuery({ ...query, question: e.target.value })} placeholder="描述想分析的问题，如 9 点后支付接口超时增多，可能是什么原因？" required />
            <div className="flex gap-2">
              <Button disabled={!query.sourceId || previewMutation.isPending}>
                <Search className="h-4 w-4" />
                {previewMutation.isPending ? '预览中...' : '预览日志'}
              </Button>
              <Button type="button" className="bg-amber-700 hover:bg-amber-800" disabled={!query.sourceId || !query.question || analysisMutation.isPending} onClick={analyze}>
                <Sparkles className="h-4 w-4" />
                {analysisMutation.isPending ? '分析中...' : '开始分析'}
              </Button>
            </div>
          </form>
        </Card>

        <div className="space-y-4">
          {previewMutation.error && <ErrorCard message={previewMutation.error instanceof Error ? previewMutation.error.message : '日志预览失败'} />}
          {analysisMutation.error && <ErrorCard message={analysisMutation.error instanceof Error ? analysisMutation.error.message : '日志分析失败'} />}
          {analysisMutation.data && (
            <Card className="p-4">
              <h2 className="text-sm font-semibold">分析结果</h2>
              <p className="mt-2 text-sm text-slate-700">{analysisMutation.data.summary}</p>
              <ResultList title="日志证据" items={analysisMutation.data.evidence} />
              <ResultList title="可能原因" items={analysisMutation.data.possibleCauses} />
              <ResultList title="排查建议" items={analysisMutation.data.suggestions} />
              <ResultList title="风险提示" items={analysisMutation.data.riskTips} />
              {analysisMutation.data.citations?.length > 0 && <CitationList citations={analysisMutation.data.citations} />}
            </Card>
          )}
          {previewMutation.data && (
            <Card className="p-4">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold">日志样本</h2>
                <span className="text-xs text-slate-500">{previewMutation.data.total} 条</span>
              </div>
              <div className="mt-3 space-y-2">
                {previewMutation.data.items.map((item, index) => <LogLine key={index} item={item} />)}
                {previewMutation.data.items.length === 0 && <div className="text-sm text-slate-500">未读取到符合条件的日志</div>}
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}

function ResultList({ title, items }: { title: string; items: string[] }) {
  if (!items?.length) return null
  return (
    <div className="mt-4">
      <h3 className="text-sm font-medium">{title}</h3>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-slate-700">
        {items.map((item, index) => <li key={index}>{item}</li>)}
      </ul>
    </div>
  )
}

function LogLine({ item }: { item: LogItem }) {
  return (
    <div className="rounded-md bg-slate-50 p-3 font-mono text-xs text-slate-700">
      <div className="mb-1 text-slate-500">{item.timestamp || '-'} {item.level || ''} {item.source || ''}</div>
      <div className="whitespace-pre-wrap break-words">{item.message || item.raw}</div>
    </div>
  )
}

function ErrorCard({ message }: { message: string }) {
  return <Card className="p-4 text-sm text-red-700">{message}</Card>
}
