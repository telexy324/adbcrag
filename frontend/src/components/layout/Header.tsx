import { LogOut } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { Button } from '../ui/Button'

export function Header() {
  const auth = useAuth()
  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-white px-6">
      <div className="text-sm font-medium text-slate-700">知识库问答仅用于辅助分析，不自动执行生产命令</div>
      <div className="flex items-center gap-3">
        <div className="text-xs text-slate-500">{auth.user?.displayName || auth.user?.username} · {auth.user?.role}</div>
        <Button className="h-8 bg-slate-700 px-3 hover:bg-slate-800" onClick={auth.logout} title="退出登录">
          <LogOut className="h-4 w-4" />
        </Button>
      </div>
    </header>
  )
}
