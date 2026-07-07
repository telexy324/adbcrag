package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/security"

	"gorm.io/datatypes"
)

const (
	k8sAuthTypeBearerToken = "bearer_token"
	k8sDiagPod             = "pod"
	k8sDiagAlert           = "alert"
	k8sDiagIngress         = "ingress"
	k8sDiagService         = "service"
)

type K8sCredentials struct {
	BearerToken string `json:"bearerToken,omitempty"`
	CACert      string `json:"caCert,omitempty"`
}

type K8sService struct {
	cfg      *config.Config
	clusters *repository.K8sClusterRepository
	tasks    *repository.K8sDiagnosisRepository
	chunks   *repository.ChunkRepository
	crypto   *security.CredentialCrypto
	k8s      client.K8sClient
	llm      client.DeepSeekClient
	masks    []*regexp.Regexp
}

func NewK8sService(cfg *config.Config, clusters *repository.K8sClusterRepository, tasks *repository.K8sDiagnosisRepository, chunks *repository.ChunkRepository, crypto *security.CredentialCrypto, k8s client.K8sClient, llm client.DeepSeekClient) *K8sService {
	return &K8sService{
		cfg: cfg, clusters: clusters, tasks: tasks, chunks: chunks, crypto: crypto, k8s: k8s, llm: llm,
		masks: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|access[_-]?key|authorization|cookie)[=:]\s*[^,\s]+`),
			regexp.MustCompile(`\b1[3-9]\d{9}\b`),
			regexp.MustCompile(`\b\d{16,19}\b`),
			regexp.MustCompile(`\b\d{17}[\dXx]\b`),
		},
	}
}

func (s *K8sService) ListClusters(ctx context.Context) ([]model.K8sCluster, error) {
	return s.clusters.List(ctx)
}

func (s *K8sService) CreateCluster(ctx context.Context, req dto.SaveK8sClusterRequest, createdBy string) (*model.K8sCluster, error) {
	cluster, err := s.buildCluster(req, nil)
	if err != nil {
		return nil, err
	}
	cluster.CreatedBy = createdBy
	if err := s.clusters.Create(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func (s *K8sService) UpdateCluster(ctx context.Context, id uint64, req dto.SaveK8sClusterRequest) (*model.K8sCluster, error) {
	existing, err := s.clusters.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	next, err := s.buildCluster(req, existing)
	if err != nil {
		return nil, err
	}
	next.ID = existing.ID
	next.CreatedBy = existing.CreatedBy
	next.CreatedAt = existing.CreatedAt
	if err := s.clusters.Update(ctx, next); err != nil {
		return nil, err
	}
	return next, nil
}

func (s *K8sService) DeleteCluster(ctx context.Context, id uint64) error {
	return s.clusters.Delete(ctx, id)
}

func (s *K8sService) TestCluster(ctx context.Context, id uint64) error {
	cluster, credentials, err := s.getClusterWithCredentials(ctx, id)
	if err != nil {
		return err
	}
	return s.k8s.Test(ctx, s.clientConfig(cluster, credentials))
}

func (s *K8sService) DiagnosePod(ctx context.Context, req dto.K8sPodDiagnosisRequest, createdBy string) (*dto.K8sDiagnosisResponse, error) {
	cluster, credentials, err := s.getClusterWithCredentials(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNamespaceAllowed(cluster, req.Namespace); err != nil {
		return nil, err
	}
	task := s.newTask(cluster, k8sDiagPod, req.Namespace, "Pod", req.Pod, req.Container, req.Question, createdBy)
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}
	contextData, err := s.k8s.GetPodContext(ctx, s.clientConfig(cluster, credentials), req.Namespace, req.Pod, req.Container, s.cfg.K8sLogTailLines, true)
	return s.finishDiagnosis(ctx, task, err, k8sDiagPod, req.Question, req.SystemName, req.ComponentName, req.TopK, contextData)
}

func (s *K8sService) DiagnoseAlert(ctx context.Context, req dto.K8sAlertDiagnosisRequest, createdBy string) (*dto.K8sDiagnosisResponse, error) {
	req = normalizeAlertRequest(req)
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if req.Question == "" {
		req.Question = strings.TrimSpace(req.AlertName + " " + req.Summary + " " + req.Description)
	}
	if req.Pod != "" {
		return s.DiagnosePod(ctx, dto.K8sPodDiagnosisRequest{
			ClusterID: req.ClusterID, Namespace: req.Namespace, Pod: req.Pod, Container: req.Container,
			Question: req.Question, SystemName: req.SystemName, ComponentName: req.ComponentName, TopK: req.TopK,
		}, createdBy)
	}
	if req.Ingress != "" {
		return s.DiagnoseIngress(ctx, dto.K8sResourceDiagnosisRequest{ClusterID: req.ClusterID, Namespace: req.Namespace, Name: req.Ingress, Question: req.Question, SystemName: req.SystemName, ComponentName: req.ComponentName, TopK: req.TopK}, createdBy)
	}
	if req.Service != "" {
		return s.DiagnoseService(ctx, dto.K8sResourceDiagnosisRequest{ClusterID: req.ClusterID, Namespace: req.Namespace, Name: req.Service, Question: req.Question, SystemName: req.SystemName, ComponentName: req.ComponentName, TopK: req.TopK}, createdBy)
	}
	return nil, fmt.Errorf("alert must include pod, ingress or service")
}

func (s *K8sService) DiagnoseIngress(ctx context.Context, req dto.K8sResourceDiagnosisRequest, createdBy string) (*dto.K8sDiagnosisResponse, error) {
	return s.diagnoseResource(ctx, req, createdBy, k8sDiagIngress, "Ingress", s.k8s.GetIngressContext)
}

func (s *K8sService) DiagnoseService(ctx context.Context, req dto.K8sResourceDiagnosisRequest, createdBy string) (*dto.K8sDiagnosisResponse, error) {
	return s.diagnoseResource(ctx, req, createdBy, k8sDiagService, "Service", s.k8s.GetServiceContext)
}

func (s *K8sService) diagnoseResource(ctx context.Context, req dto.K8sResourceDiagnosisRequest, createdBy, diagType, kind string, collect func(context.Context, client.K8sConfig, string, string) (map[string]any, error)) (*dto.K8sDiagnosisResponse, error) {
	cluster, credentials, err := s.getClusterWithCredentials(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureNamespaceAllowed(cluster, req.Namespace); err != nil {
		return nil, err
	}
	task := s.newTask(cluster, diagType, req.Namespace, kind, req.Name, "", req.Question, createdBy)
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}
	contextData, err := collect(ctx, s.clientConfig(cluster, credentials), req.Namespace, req.Name)
	return s.finishDiagnosis(ctx, task, err, diagType, req.Question, req.SystemName, req.ComponentName, req.TopK, contextData)
}

func (s *K8sService) finishDiagnosis(ctx context.Context, task *model.K8sDiagnosisTask, collectErr error, diagType, question, systemName, componentName string, topK int, contextData map[string]any) (*dto.K8sDiagnosisResponse, error) {
	if collectErr != nil {
		task.Status = model.LogAnalysisStatusFailed
		task.ErrorMessage = collectErr.Error()
		_ = s.tasks.Update(ctx, task)
		return nil, collectErr
	}
	contextData = s.sanitizeContext(contextData).(map[string]any)
	contextJSON, _ := json.Marshal(contextData)
	task.Context = datatypes.JSON(contextJSON)
	query := strings.TrimSpace(question + " " + diagType + " " + task.ResourceKind + " " + task.ResourceName + " " + task.Namespace + " " + compactContextKeywords(contextData))
	results, err := s.chunks.KeywordSearch(ctx, repository.SearchFilter{
		SystemName: systemName, ComponentName: componentName, Query: query,
		Keywords: uniqueNonEmpty([]string{question, diagType, task.ResourceKind, task.ResourceName, task.Namespace, systemName, componentName}),
		TopK:     choosePositive(topK, s.cfg.RAGTopK),
	})
	if err != nil {
		task.Status = model.LogAnalysisStatusFailed
		task.ErrorMessage = err.Error()
		_ = s.tasks.Update(ctx, task)
		return nil, err
	}
	citations := searchResultsToCitations(results)
	response := s.analyzeK8sWithLLM(ctx, diagType, question, contextData, results, citations)
	response.TaskID = task.ID
	response.Status = model.LogAnalysisStatusSuccess
	response.Context = contextData
	resultJSON, _ := json.Marshal(response)
	citationJSON, _ := json.Marshal(citations)
	task.Status = model.LogAnalysisStatusSuccess
	task.Result = datatypes.JSON(resultJSON)
	task.RetrievedChunks = datatypes.JSON(citationJSON)
	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *K8sService) analyzeK8sWithLLM(ctx context.Context, diagType, question string, contextData map[string]any, chunks []repository.SearchResult, citations []dto.Citation) *dto.K8sDiagnosisResponse {
	prompt := buildK8sPrompt(diagType, question, contextData, chunks)
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return fallbackK8sDiagnosis(contextData, citations)
	}
	var parsed dto.K8sDiagnosisResponse
	if unmarshalJSON(resp.Content, &parsed) != nil {
		parsed = *fallbackK8sDiagnosis(contextData, citations)
	} else {
		parsed.Citations = citations
	}
	if len(parsed.RiskTips) == 0 {
		parsed.RiskTips = []string{"AI K8s 分析仅供运维排查参考；禁止自动删除、重启、扩缩容或修改 Kubernetes 资源。"}
	}
	return &parsed
}

func buildK8sPrompt(diagType, question string, contextData map[string]any, chunks []repository.SearchResult) string {
	contextJSON, _ := json.MarshalIndent(contextData, "", "  ")
	var cb strings.Builder
	for i, chunk := range chunks {
		fmt.Fprintf(&cb, "引用 %d：文档《%s》章节「%s」\n%s\n\n", i+1, chunk.DocumentTitle, chunk.SourceSection, truncate(chunk.Content, 1000))
	}
	return fmt.Sprintf(`你是一个资深银行生产运维 Kubernetes 诊断专家。

请基于【K8s 只读上下文】和【知识库内容】分析用户问题，只输出 JSON。

要求：
1. 区分 K8s 资源事实、日志证据、知识库依据和推测原因。
2. 不要编造上下文中不存在的 Pod、Event、Service、Endpoint、Ingress、时间点或日志。
3. 只允许给排查建议，不允许建议系统自动执行 kubectl、delete、restart、scale、patch、apply、edit。
4. 涉及删除、重启、扩缩容、回滚、修改配置等高风险操作，必须提示走生产变更审批。

输出格式：
{"summary":"","possibleCauses":[],"evidence":[],"suggestions":[],"riskTips":[]}

诊断类型：
%s

用户问题：
%s

K8s 只读上下文：
%s

知识库内容：
%s`, diagType, question, string(contextJSON), cb.String())
}

func fallbackK8sDiagnosis(contextData map[string]any, citations []dto.Citation) *dto.K8sDiagnosisResponse {
	contextJSON, _ := json.Marshal(contextData)
	return &dto.K8sDiagnosisResponse{
		Summary:     "已采集 Kubernetes 只读上下文，但 LLM 分析不可用或返回格式异常；请结合关键证据人工复核。",
		Evidence:    []string{truncate(string(contextJSON), 1200)},
		Suggestions: []string{"优先检查 Pod 状态、containerStatuses、Events、previous logs、Service selector 与 Endpoint 就绪情况。"},
		RiskTips:    []string{"AI K8s 分析仅供运维排查参考；禁止自动删除、重启、扩缩容或修改 Kubernetes 资源。"},
		Citations:   citations,
	}
}

func (s *K8sService) buildCluster(req dto.SaveK8sClusterRequest, existing *model.K8sCluster) (*model.K8sCluster, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.ClusterCode = strings.TrimSpace(req.ClusterCode)
	req.APIServer = strings.TrimRight(strings.TrimSpace(req.APIServer), "/")
	if req.Name == "" || req.ClusterCode == "" || req.APIServer == "" {
		return nil, fmt.Errorf("name, clusterCode and apiServer are required")
	}
	if req.AuthType == "" {
		req.AuthType = k8sAuthTypeBearerToken
	}
	if req.AuthType != k8sAuthTypeBearerToken {
		return nil, fmt.Errorf("unsupported authType: %s", req.AuthType)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else if existing != nil {
		enabled = existing.Enabled
	}
	allowed, _ := json.Marshal(uniqueNonEmpty(req.AllowedNamespaces))
	cluster := &model.K8sCluster{
		Name: req.Name, ClusterCode: req.ClusterCode, APIServer: req.APIServer, AuthType: req.AuthType,
		AllowedNamespaces: datatypes.JSON(allowed), InsecureSkipTLSVerify: req.InsecureSkipTLSVerify, Enabled: enabled,
	}
	if existing != nil && req.BearerToken == "" && req.CACert == "" {
		cluster.CredentialRef = existing.CredentialRef
		return cluster, nil
	}
	ref, err := s.encryptCredentials(K8sCredentials{BearerToken: req.BearerToken, CACert: req.CACert})
	if err != nil {
		return nil, err
	}
	cluster.CredentialRef = ref
	return cluster, nil
}

func (s *K8sService) getClusterWithCredentials(ctx context.Context, id uint64) (*model.K8sCluster, K8sCredentials, error) {
	cluster, err := s.clusters.GetByID(ctx, id)
	if err != nil {
		return nil, K8sCredentials{}, err
	}
	if !cluster.Enabled {
		return nil, K8sCredentials{}, fmt.Errorf("k8s cluster is disabled")
	}
	credentials, err := s.decryptCredentials(cluster.CredentialRef)
	return cluster, credentials, err
}

func (s *K8sService) clientConfig(cluster *model.K8sCluster, credentials K8sCredentials) client.K8sConfig {
	return client.K8sConfig{
		APIServer: cluster.APIServer, BearerToken: credentials.BearerToken, CACert: credentials.CACert,
		InsecureSkipTLSVerify: cluster.InsecureSkipTLSVerify, Timeout: time.Duration(s.cfg.ESTimeoutSec) * time.Second,
	}
}

func (s *K8sService) ensureNamespaceAllowed(cluster *model.K8sCluster, namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	allowed := decodeStringArray(cluster.AllowedNamespaces)
	if len(allowed) == 0 {
		return nil
	}
	for _, item := range allowed {
		if item == namespace {
			return nil
		}
	}
	return fmt.Errorf("namespace %s is not allowed for cluster %s", namespace, cluster.ClusterCode)
}

func (s *K8sService) newTask(cluster *model.K8sCluster, diagType, namespace, kind, name, container, question, createdBy string) *model.K8sDiagnosisTask {
	return &model.K8sDiagnosisTask{
		ClusterID: cluster.ID, ClusterCode: cluster.ClusterCode, DiagnosisType: diagType, Namespace: namespace,
		ResourceKind: kind, ResourceName: name, ContainerName: container, Question: question,
		Status: model.LogAnalysisStatusRunning, CreatedBy: createdBy,
	}
}

func (s *K8sService) encryptCredentials(credentials K8sCredentials) (string, error) {
	data, _ := json.Marshal(credentials)
	return s.crypto.Encrypt(string(data))
}

func (s *K8sService) decryptCredentials(ref string) (K8sCredentials, error) {
	raw, err := s.crypto.Decrypt(ref)
	if err != nil {
		return K8sCredentials{}, err
	}
	var credentials K8sCredentials
	if raw != "" {
		err = json.Unmarshal([]byte(raw), &credentials)
	}
	return credentials, err
}

func (s *K8sService) sanitizeContext(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range typed {
			lower := strings.ToLower(key)
			if lower == "managedfields" {
				continue
			}
			if lower == "env" {
				out[key] = sanitizeEnvKeys(item)
				continue
			}
			out[key] = s.sanitizeContext(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, s.sanitizeContext(item))
		}
		return out
	case string:
		text := typed
		for _, mask := range s.masks {
			text = mask.ReplaceAllString(text, "[MASKED]")
		}
		return truncate(text, s.cfg.K8sLogMaxBytes)
	default:
		return value
	}
}

func sanitizeEnvKeys(value any) any {
	items, ok := value.([]any)
	if !ok {
		return value
	}
	out := []any{}
	for _, item := range items {
		env, _ := item.(map[string]any)
		if env == nil {
			continue
		}
		next := map[string]any{}
		if name, ok := env["name"]; ok {
			next["name"] = name
		}
		if _, ok := env["valueFrom"]; ok {
			next["valueFrom"] = env["valueFrom"]
		}
		out = append(out, next)
	}
	return out
}

func normalizeAlertRequest(req dto.K8sAlertDiagnosisRequest) dto.K8sAlertDiagnosisRequest {
	if req.Labels != nil {
		req.AlertName = chooseNonEmpty(req.AlertName, req.Labels["alertname"], req.Labels["alertName"])
		req.Namespace = chooseNonEmpty(req.Namespace, req.Labels["namespace"])
		req.Pod = chooseNonEmpty(req.Pod, req.Labels["pod"])
		req.Container = chooseNonEmpty(req.Container, req.Labels["container"])
		req.Deployment = chooseNonEmpty(req.Deployment, req.Labels["deployment"])
		req.Service = chooseNonEmpty(req.Service, req.Labels["service"])
		req.Ingress = chooseNonEmpty(req.Ingress, req.Labels["ingress"])
		req.Node = chooseNonEmpty(req.Node, req.Labels["node"])
		req.Severity = chooseNonEmpty(req.Severity, req.Labels["severity"])
	}
	if req.Annotations != nil {
		req.Summary = chooseNonEmpty(req.Summary, req.Annotations["summary"])
		req.Description = chooseNonEmpty(req.Description, req.Annotations["description"])
	}
	return req
}

func compactContextKeywords(contextData map[string]any) string {
	data, _ := json.Marshal(contextData)
	re := regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.:/-]{3,}`)
	words := re.FindAllString(string(data), 80)
	return strings.Join(uniqueNonEmpty(words), " ")
}
