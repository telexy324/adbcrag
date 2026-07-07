package dto

type SaveK8sClusterRequest struct {
	Name                  string   `json:"name" binding:"required"`
	ClusterCode           string   `json:"clusterCode" binding:"required"`
	APIServer             string   `json:"apiServer" binding:"required"`
	AuthType              string   `json:"authType"`
	BearerToken           string   `json:"bearerToken"`
	CACert                string   `json:"caCert"`
	AllowedNamespaces     []string `json:"allowedNamespaces"`
	InsecureSkipTLSVerify bool     `json:"insecureSkipTLSVerify"`
	Enabled               *bool    `json:"enabled"`
}

type TestK8sClusterResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type K8sPodDiagnosisRequest struct {
	ClusterID     uint64 `json:"clusterId" binding:"required"`
	Namespace     string `json:"namespace" binding:"required"`
	Pod           string `json:"pod" binding:"required"`
	Container     string `json:"container"`
	Question      string `json:"question"`
	SystemName    string `json:"systemName"`
	ComponentName string `json:"componentName"`
	TopK          int    `json:"topK"`
}

type K8sAlertDiagnosisRequest struct {
	ClusterID     uint64            `json:"clusterId" binding:"required"`
	AlertName     string            `json:"alertName"`
	Namespace     string            `json:"namespace"`
	Pod           string            `json:"pod"`
	Container     string            `json:"container"`
	Deployment    string            `json:"deployment"`
	Service       string            `json:"service"`
	Ingress       string            `json:"ingress"`
	Node          string            `json:"node"`
	Severity      string            `json:"severity"`
	Summary       string            `json:"summary"`
	Description   string            `json:"description"`
	Labels        map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	Question      string            `json:"question"`
	SystemName    string            `json:"systemName"`
	ComponentName string            `json:"componentName"`
	TopK          int               `json:"topK"`
}

type K8sResourceDiagnosisRequest struct {
	ClusterID     uint64 `json:"clusterId" binding:"required"`
	Namespace     string `json:"namespace" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Question      string `json:"question"`
	SystemName    string `json:"systemName"`
	ComponentName string `json:"componentName"`
	TopK          int    `json:"topK"`
}

type K8sDiagnosisResponse struct {
	TaskID         uint64     `json:"taskId"`
	Status         string     `json:"status"`
	Summary        string     `json:"summary"`
	PossibleCauses []string   `json:"possibleCauses"`
	Evidence       []string   `json:"evidence"`
	Suggestions    []string   `json:"suggestions"`
	RiskTips       []string   `json:"riskTips"`
	Citations      []Citation `json:"citations"`
	Context        any        `json:"context,omitempty"`
}
