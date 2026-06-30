import { Link } from 'react-router-dom'
import { type DocumentItem } from '../../api/documentApi'
import { DocumentStatusBadge } from './DocumentStatusBadge'

export function DocumentTable({ items }: { items: DocumentItem[] }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border bg-white">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted text-slate-600">
          <tr>
            <th className="px-4 py-3">标题</th>
            <th className="px-4 py-3">系统</th>
            <th className="px-4 py-3">组件</th>
            <th className="px-4 py-3">类型</th>
            <th className="px-4 py-3">状态</th>
            <th className="px-4 py-3">质量分</th>
          </tr>
        </thead>
        <tbody>
          {items.map((doc) => (
            <tr key={doc.id} className="border-t border-border">
              <td className="px-4 py-3 font-medium text-teal-800"><Link to={`/documents/${doc.id}`}>{doc.title}</Link></td>
              <td className="px-4 py-3">{doc.systemName || '-'}</td>
              <td className="px-4 py-3">{doc.componentName || '-'}</td>
              <td className="px-4 py-3">{doc.docType || '-'}</td>
              <td className="px-4 py-3"><DocumentStatusBadge status={doc.status} /></td>
              <td className="px-4 py-3">{doc.qualityScore}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
