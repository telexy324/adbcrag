import { ClipboardList, FileCheck, FileText, Home, MessageSquare, SearchCode, ServerCog, Upload } from 'lucide-react'
import { NavLink } from 'react-router-dom'
import { cn } from '../../lib/utils'

const items = [
  { to: '/', label: '概览', icon: Home },
  { to: '/documents', label: '文档库', icon: FileText },
  { to: '/upload', label: '文档入库', icon: Upload },
  { to: '/quality-criteria', label: '评分标准', icon: ClipboardList },
  { to: '/log-sources', label: '日志源', icon: ServerCog },
  { to: '/log-analysis', label: '日志分析', icon: SearchCode },
  { to: '/chat', label: '知识问答', icon: MessageSquare },
  { to: '/review', label: '审核发布', icon: FileCheck },
]

export function Sidebar() {
  return (
    <aside className="flex h-screen w-64 flex-col border-r border-border bg-white">
      <div className="border-b border-border px-5 py-4">
        <div className="text-base font-semibold">运维知识库 RAG</div>
        <div className="mt-1 text-xs text-slate-500">生产排查辅助系统</div>
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {items.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                cn('flex items-center gap-3 rounded-md px-3 py-2 text-sm text-slate-700 hover:bg-muted', isActive && 'bg-teal-50 text-teal-800')
              }
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </NavLink>
          )
        })}
      </nav>
    </aside>
  )
}
