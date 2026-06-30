import { DocumentUploadForm } from '../components/document/DocumentUploadForm'

export function DocumentUploadPage() {
  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">文档入库</h1>
      <DocumentUploadForm />
    </div>
  )
}
