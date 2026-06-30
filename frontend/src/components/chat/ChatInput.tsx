import { FormEvent } from 'react'
import { Button } from '../ui/Button'
import { Input } from '../ui/Input'
import { Textarea } from '../ui/Textarea'

export function ChatInput({ loading, onAsk }: { loading: boolean; onAsk: (data: any) => void }) {
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    onAsk(Object.fromEntries(new FormData(event.currentTarget).entries()))
  }
  return (
    <form className="space-y-3" onSubmit={submit}>
      <Textarea name="question" placeholder="Redis 内存告警怎么处理？" required />
      <div className="grid gap-3 md:grid-cols-[1fr_1fr_120px_auto]">
        <Input name="systemName" placeholder="系统过滤" />
        <Input name="componentName" placeholder="组件过滤" />
        <Input name="topK" type="number" min={1} max={20} defaultValue={5} />
        <Button disabled={loading}>{loading ? '检索中...' : '提问'}</Button>
      </div>
    </form>
  )
}
