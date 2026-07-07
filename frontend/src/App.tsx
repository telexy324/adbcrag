import { Navigate, Route, Routes } from 'react-router-dom'
import { RequireAuth } from './auth/AuthContext'
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
import { LoginPage } from './pages/LoginPage'
import { QualityCriteriaPage } from './pages/QualityCriteriaPage'
import { ReviewPage } from './pages/ReviewPage'
import { UserManagePage } from './pages/UserManagePage'

export function App() {
  return (
    <Routes>
      <Route path="login" element={<LoginPage />} />
      <Route element={<RequireAuth><AppLayout /></RequireAuth>}>
        <Route index element={<DashboardPage />} />
        <Route path="documents" element={<DocumentListPage />} />
        <Route path="documents/:id" element={<DocumentDetailPage />} />
        <Route path="upload" element={<DocumentUploadPage />} />
        <Route path="quality-criteria" element={<RequireAuth adminOnly><QualityCriteriaPage /></RequireAuth>} />
        <Route path="llm-configs" element={<RequireAuth adminOnly><LLMConfigPage /></RequireAuth>} />
        <Route path="log-sources" element={<RequireAuth adminOnly><LogSourcePage /></RequireAuth>} />
        <Route path="log-analysis" element={<LogAnalysisPage />} />
        <Route path="k8s-clusters" element={<RequireAuth adminOnly><K8sClusterPage /></RequireAuth>} />
        <Route path="k8s-diagnosis" element={<K8sDiagnosisPage />} />
        <Route path="chat" element={<ChatPage />} />
        <Route path="review" element={<RequireAuth adminOnly><ReviewPage /></RequireAuth>} />
        <Route path="users" element={<RequireAuth adminOnly><UserManagePage /></RequireAuth>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}
