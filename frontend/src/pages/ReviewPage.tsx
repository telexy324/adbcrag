import { ReviewPanel } from '../components/review/ReviewPanel'

export function ReviewPage() {
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">文档审核</h1>
      <ReviewPanel />
    </div>
  )
}
