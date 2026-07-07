import { Navigate, Route, Routes } from 'react-router-dom'
import { AppLayout } from './components/layout/AppLayout'
import { ChatPage } from './pages/ChatPage'
import { DashboardPage } from './pages/DashboardPage'
import { DocumentDetailPage } from './pages/DocumentDetailPage'
import { DocumentListPage } from './pages/DocumentListPage'
import { DocumentUploadPage } from './pages/DocumentUploadPage'
import { LogAnalysisPage } from './pages/LogAnalysisPage'
import { LogSourcePage } from './pages/LogSourcePage'
import { LLMConfigPage } from './pages/LLMConfigPage'
import { K8sClusterPage } from './pages/K8sClusterPage'
import { K8sDiagnosisPage } from './pages/K8sDiagnosisPage'
import { QualityCriteriaPage } from './pages/QualityCriteriaPage'
import { ReviewPage } from './pages/ReviewPage'

export function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="documents" element={<DocumentListPage />} />
        <Route path="documents/:id" element={<DocumentDetailPage />} />
        <Route path="upload" element={<DocumentUploadPage />} />
        <Route path="quality-criteria" element={<QualityCriteriaPage />} />
        <Route path="llm-configs" element={<LLMConfigPage />} />
        <Route path="log-sources" element={<LogSourcePage />} />
        <Route path="log-analysis" element={<LogAnalysisPage />} />
        <Route path="k8s-clusters" element={<K8sClusterPage />} />
        <Route path="k8s-diagnosis" element={<K8sDiagnosisPage />} />
        <Route path="chat" element={<ChatPage />} />
        <Route path="review" element={<ReviewPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
