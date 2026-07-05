export function Header() {
  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-white px-6">
      <div className="text-sm font-medium text-slate-700">知识库问答仅用于辅助分析，不自动执行生产命令</div>
      <div className="text-xs text-slate-500">可选大模型 · pg_trgm</div>
    </header>
  )
}
