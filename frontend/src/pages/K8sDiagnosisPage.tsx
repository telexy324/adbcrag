import { useMutation, useQuery } from '@tanstack/react-query'
import { FormEvent, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { diagnoseK8sAlert, diagnoseK8sIngress, diagnoseK8sPod, diagnoseK8sService, K8sDiagnosisResult, listK8sClusters } from '../api/k8sApi'
import { CitationList } from '../components/chat/CitationList'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { Input } from '../components/ui/Input'
import { Select } from '../components/ui/Select'
import { Textarea } from '../components/ui/Textarea'

type DiagnosisType = 'pod' | 'alert' | 'ingress' | 'service'

type FormState = {
  type: DiagnosisType
  alertResourceKind: 'pod' | 'ingress' | 'service'
  clusterId: string
  namespace: string
  name: string
  pod: string
  container: string
  alertName: string
  severity: string
  summary: string
  description: string
  question: string
  systemName: string
  componentName: string
}

const emptyForm: FormState = {
  type: 'pod',
  alertResourceKind: 'pod',
  clusterId: '',
  namespace: '',
  name: '',
  pod: '',
  container: '',
  alertName: '',
  severity: '',
  summary: '',
  description: '',
  question: '',
  systemName: '',
  componentName: '',
}

export function K8sDiagnosisPage() {
  const [form, setForm] = useState<FormState>(emptyForm)
  const { data: clusters = [] } = useQuery({ queryKey: ['k8s-clusters'], queryFn: listK8sClusters })
  const diagnosisMutation = useMutation({ mutationFn: submitDiagnosis })

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    diagnosisMutation.mutate(form)
  }

  return (
    <div className="space-y-5">
      <h1 className="text-xl font-semibold">K8s 诊断</h1>
      <div className="grid gap-5 xl:grid-cols-[460px_minmax(0,1fr)]">
        <Card className="p-5">
          <form className="grid gap-4" onSubmit={submit}>
            <Select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value as DiagnosisType })}>
              <option value="pod">Pod 诊断</option>
              <option value="alert">告警诊断</option>
              <option value="ingress">Ingress 诊断</option>
              <option value="service">Service 诊断</option>
            </Select>
            <Select value={form.clusterId} onChange={(e) => setForm({ ...form, clusterId: e.target.value })} required>
              <option value="">选择 K8s 集群</option>
              {clusters.map((cluster) => (
                <option key={cluster.id} value={cluster.id}>{cluster.name} · {cluster.clusterCode}</option>
              ))}
            </Select>
            <Input value={form.namespace} onChange={(e) => setForm({ ...form, namespace: e.target.value })} placeholder="namespace" required />

            {form.type === 'pod' && (
              <>
                <Input value={form.pod} onChange={(e) => setForm({ ...form, pod: e.target.value })} placeholder="Pod 名称" required />
                <Input value={form.container} onChange={(e) => setForm({ ...form, container: e.target.value })} placeholder="容器名，可选" />
              </>
            )}

            {(form.type === 'ingress' || form.type === 'service') && (
              <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={`${form.type === 'ingress' ? 'Ingress' : 'Service'} 名称`} required />
            )}

            {form.type === 'alert' && (
              <>
                <div className="grid gap-3 md:grid-cols-2">
                  <Input value={form.alertName} onChange={(e) => setForm({ ...form, alertName: e.target.value })} placeholder="alertname" />
                  <Input value={form.severity} onChange={(e) => setForm({ ...form, severity: e.target.value })} placeholder="severity" />
                </div>
                <Select value={form.alertResourceKind} onChange={(e) => setForm({ ...form, alertResourceKind: e.target.value as FormState['alertResourceKind'] })}>
                  <option value="pod">Pod</option>
                  <option value="ingress">Ingress</option>
                  <option value="service">Service</option>
                </Select>
                {form.alertResourceKind === 'pod' ? (
                  <>
                    <Input value={form.pod} onChange={(e) => setForm({ ...form, pod: e.target.value })} placeholder="Pod 名称" />
                    <Input value={form.container} onChange={(e) => setForm({ ...form, container: e.target.value })} placeholder="容器名，可选" />
                  </>
                ) : (
                  <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder={`${form.alertResourceKind === 'ingress' ? 'Ingress' : 'Service'} 名称`} />
                )}
                <Textarea value={form.summary} onChange={(e) => setForm({ ...form, summary: e.target.value })} placeholder="告警摘要" />
                <Textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="告警描述" />
              </>
            )}

            <div className="grid gap-3 md:grid-cols-2">
              <Input value={form.systemName} onChange={(e) => setForm({ ...form, systemName: e.target.value })} placeholder="系统，可选" />
              <Input value={form.componentName} onChange={(e) => setForm({ ...form, componentName: e.target.value })} placeholder="组件，可选" />
            </div>
            <Textarea value={form.question} onChange={(e) => setForm({ ...form, question: e.target.value })} placeholder="补充问题，如 最近 30 分钟频繁重启，优先排查什么？" />
            <Button disabled={!form.clusterId || !form.namespace || diagnosisMutation.isPending}>
              <Sparkles className="h-4 w-4" />
              {diagnosisMutation.isPending ? '诊断中...' : '开始诊断'}
            </Button>
          </form>
        </Card>

        <div className="space-y-4">
          {diagnosisMutation.error && <Card className="p-4 text-sm text-red-700">{diagnosisMutation.error instanceof Error ? diagnosisMutation.error.message : '诊断失败'}</Card>}
          {diagnosisMutation.data && <DiagnosisResult result={diagnosisMutation.data} />}
        </div>
      </div>
    </div>
  )
}

function submitDiagnosis(form: FormState) {
  const base = {
    clusterId: Number(form.clusterId),
    namespace: form.namespace,
    question: form.question,
    systemName: form.systemName,
    componentName: form.componentName,
  }
  if (form.type === 'pod') {
    return diagnoseK8sPod({ ...base, pod: form.pod, container: form.container })
  }
  if (form.type === 'ingress') {
    return diagnoseK8sIngress({ ...base, name: form.name })
  }
  if (form.type === 'service') {
    return diagnoseK8sService({ ...base, name: form.name })
  }
  return diagnoseK8sAlert({
    ...base,
    alertName: form.alertName,
    severity: form.severity,
    summary: form.summary,
    description: form.description,
    pod: form.pod,
    container: form.container,
    name: form.name,
    labels: {
      alertname: form.alertName,
      namespace: form.namespace,
      pod: form.pod,
      container: form.container,
      severity: form.severity,
      service: form.alertResourceKind === 'service' ? form.name : '',
      ingress: form.alertResourceKind === 'ingress' ? form.name : '',
    },
  })
}

function DiagnosisResult({ result }: { result: K8sDiagnosisResult }) {
  return (
    <Card className="p-4">
      <h2 className="text-sm font-semibold">诊断结果</h2>
      <p className="mt-2 text-sm text-slate-700">{result.summary}</p>
      <ResultList title="关键证据" items={result.evidence} />
      <ResultList title="可能原因" items={result.possibleCauses} />
      <ResultList title="排查建议" items={result.suggestions} />
      <ResultList title="风险提示" items={result.riskTips} />
      {result.citations?.length > 0 && <CitationList citations={result.citations} />}
      {result.context != null && (
        <details className="mt-4">
          <summary className="cursor-pointer text-sm font-medium text-slate-700">K8s 只读上下文</summary>
          <pre className="mt-2 max-h-[520px] overflow-auto rounded-md bg-slate-950 p-3 text-xs text-slate-100">{JSON.stringify(result.context, null, 2)}</pre>
        </details>
      )}
    </Card>
  )
}

function ResultList({ title, items }: { title: string; items: string[] }) {
  if (!items?.length) return null
  return (
    <div className="mt-4">
      <h3 className="text-sm font-medium">{title}</h3>
      <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-slate-700">
        {items.map((item, index) => <li key={index}>{item}</li>)}
      </ul>
    </div>
  )
}
