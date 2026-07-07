import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { askQuestion } from '../../api/qaApi'
import { archiveConversation, createConversation, getConversationMessages, listConversations } from '../../api/conversationApi'
import { Button } from '../ui/Button'
import { Card } from '../ui/Card'
import { AnswerCard } from './AnswerCard'
import { ChatInput } from './ChatInput'
import { CitationList } from './CitationList'

export function ChatWindow() {
  const queryClient = useQueryClient()
  const [conversationId, setConversationId] = useState<number | undefined>()
  const [loading, setLoading] = useState(false)
  const [answer, setAnswer] = useState('')
  const [citations, setCitations] = useState<any[]>([])
  const [error, setError] = useState('')
  const { data: conversations = [] } = useQuery({ queryKey: ['conversations'], queryFn: listConversations })
  const { data: activeConversation } = useQuery({
    queryKey: ['conversation-messages', conversationId],
    queryFn: () => getConversationMessages(conversationId!),
    enabled: !!conversationId,
  })
  const createMutation = useMutation({
    mutationFn: () => createConversation({ title: '新的问答会话', conversationType: 'qa' }),
    onSuccess: (item) => {
      setConversationId(item.id)
      setAnswer('')
      setCitations([])
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
  })
  const archiveMutation = useMutation({
    mutationFn: archiveConversation,
    onSuccess: () => {
      setConversationId(undefined)
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
  })

  async function onAsk(data: any) {
    setLoading(true)
    setError('')
    try {
      const result = await askQuestion({ ...data, conversationId, topK: Number(data.topK || 5) })
      setConversationId(result.conversationId)
      setAnswer(result.answer)
      setCitations(result.citations || [])
      queryClient.invalidateQueries({ queryKey: ['conversations'] })
      queryClient.invalidateQueries({ queryKey: ['conversation-messages', result.conversationId] })
    } catch (err) {
      setError(err instanceof Error ? err.message : '问答失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="grid gap-5 xl:grid-cols-[280px_minmax(0,1fr)]">
      <Card className="p-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">会话</h2>
          <Button className="h-8 px-3" onClick={() => createMutation.mutate()}>新建</Button>
        </div>
        <div className="mt-3 space-y-2">
          {conversations.map((item) => (
            <button
              key={item.id}
              className={`block w-full rounded-md px-3 py-2 text-left text-sm ${conversationId === item.id ? 'bg-teal-50 text-teal-800' : 'hover:bg-slate-50'}`}
              onClick={() => {
                setConversationId(item.id)
                setAnswer('')
                setCitations([])
              }}
            >
              <div className="truncate font-medium">{item.title || '未命名会话'}</div>
              <div className="mt-1 text-xs text-slate-500">{new Date(item.lastMessageAt || item.createdAt).toLocaleString()}</div>
            </button>
          ))}
          {conversations.length === 0 && <div className="py-6 text-center text-sm text-slate-500">暂无会话</div>}
        </div>
        {conversationId && (
          <Button className="mt-3 h-8 w-full bg-slate-700 hover:bg-slate-800" onClick={() => archiveMutation.mutate(conversationId)}>归档当前会话</Button>
        )}
      </Card>

      <div className="space-y-5">
        {activeConversation?.messages?.length ? (
          <Card className="max-h-[420px] overflow-auto p-4">
            <div className="space-y-3">
              {activeConversation.messages.map((message) => (
                <div key={message.id} className={message.role === 'user' ? 'text-right' : 'text-left'}>
                  <div className={`inline-block max-w-[85%] rounded-md px-3 py-2 text-sm ${message.role === 'user' ? 'bg-teal-700 text-white' : 'bg-slate-100 text-slate-800'}`}>
                    <div className="whitespace-pre-wrap text-left">{message.content}</div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        ) : null}
        <ChatInput loading={loading} onAsk={onAsk} />
        {error && <div className="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</div>}
        <AnswerCard answer={answer} />
        <CitationList citations={citations} />
      </div>
    </div>
  )
}
