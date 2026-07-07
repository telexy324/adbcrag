import { http, type APIResponse } from './http'
import type { Citation } from './qaApi'

export type K8sCluster = {
  id: number
  name: string
  clusterCode: string
  apiServer: string
  authType: 'bearer_token'
  allowedNamespaces: string[] | null
  insecureSkipTLSVerify: boolean
  enabled: boolean
  updatedAt: string
}

export type K8sClusterInput = Partial<K8sCluster> & {
  name: string
  clusterCode: string
  apiServer: string
  bearerToken?: string
  caCert?: string
  allowedNamespaces?: string[]
}

export type K8sDiagnosisInput = {
  clusterId: number
  namespace: string
  pod?: string
  container?: string
  name?: string
  question?: string
  systemName?: string
  componentName?: string
  topK?: number
}

export type K8sAlertDiagnosisInput = K8sDiagnosisInput & {
  alertName?: string
  severity?: string
  summary?: string
  description?: string
  labels?: Record<string, string>
  annotations?: Record<string, string>
}

export type K8sDiagnosisResult = {
  taskId: number
  status: string
  summary: string
  possibleCauses: string[]
  evidence: string[]
  suggestions: string[]
  riskTips: string[]
  citations: Citation[]
  context?: unknown
}

export async function listK8sClusters() {
  const { data } = await http.get<APIResponse<K8sCluster[]>>('/k8s/clusters')
  return data.data
}

export async function createK8sCluster(input: K8sClusterInput) {
  const { data } = await http.post<APIResponse<K8sCluster>>('/k8s/clusters', input)
  return data.data
}

export async function updateK8sCluster(id: number, input: K8sClusterInput) {
  const { data } = await http.put<APIResponse<K8sCluster>>(`/k8s/clusters/${id}`, input)
  return data.data
}

export async function deleteK8sCluster(id: number) {
  const { data } = await http.delete<APIResponse<{ deleted: boolean }>>(`/k8s/clusters/${id}`)
  return data.data
}

export async function testK8sCluster(id: number) {
  const { data } = await http.post<APIResponse<{ ok: boolean; message: string }>>(`/k8s/clusters/${id}/test`)
  return data.data
}

export async function diagnoseK8sPod(input: K8sDiagnosisInput) {
  const { data } = await http.post<APIResponse<K8sDiagnosisResult>>('/k8s/diagnosis/pod', input)
  return data.data
}

export async function diagnoseK8sAlert(input: K8sAlertDiagnosisInput) {
  const { data } = await http.post<APIResponse<K8sDiagnosisResult>>('/k8s/diagnosis/alert', input)
  return data.data
}

export async function diagnoseK8sIngress(input: K8sDiagnosisInput) {
  const { data } = await http.post<APIResponse<K8sDiagnosisResult>>('/k8s/diagnosis/ingress', input)
  return data.data
}

export async function diagnoseK8sService(input: K8sDiagnosisInput) {
  const { data } = await http.post<APIResponse<K8sDiagnosisResult>>('/k8s/diagnosis/service', input)
  return data.data
}
