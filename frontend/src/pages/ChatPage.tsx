import { ChatWindow } from '../components/chat/ChatWindow'

export function ChatPage() {
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">知识库问答</h1>
      <ChatWindow />
    </div>
  )
}
