import { useState } from 'react'
import { askQuestion } from '../../api/qaApi'
import { AnswerCard } from './AnswerCard'
import { ChatInput } from './ChatInput'
import { CitationList } from './CitationList'

export function ChatWindow() {
  const [loading, setLoading] = useState(false)
  const [answer, setAnswer] = useState('')
  const [citations, setCitations] = useState<any[]>([])
  const [error, setError] = useState('')

  async function onAsk(data: any) {
    setLoading(true)
    setError('')
    try {
      const result = await askQuestion({ ...data, topK: Number(data.topK || 5) })
      setAnswer(result.answer)
      setCitations(result.citations || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '问答失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-5">
      <ChatInput loading={loading} onAsk={onAsk} />
      {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
      <AnswerCard answer={answer} />
      <CitationList citations={citations} />
    </div>
  )
}
