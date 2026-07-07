package model

import (
	"time"

	"gorm.io/datatypes"
)

type K8sCluster struct {
	ID                    uint64         `gorm:"primaryKey" json:"id"`
	Name                  string         `gorm:"size:120;not null" json:"name"`
	ClusterCode           string         `gorm:"size:120;not null;uniqueIndex" json:"clusterCode"`
	APIServer             string         `gorm:"type:text;not null" json:"apiServer"`
	AuthType              string         `gorm:"size:50;default:bearer_token" json:"authType"`
	CredentialRef         string         `gorm:"type:text" json:"-"`
	AllowedNamespaces     datatypes.JSON `gorm:"type:jsonb" json:"allowedNamespaces"`
	InsecureSkipTLSVerify bool           `gorm:"default:false" json:"insecureSkipTLSVerify"`
	Enabled               bool           `gorm:"default:true" json:"enabled"`
	CreatedBy             string         `gorm:"size:100" json:"createdBy"`
	CreatedAt             time.Time      `json:"createdAt"`
	UpdatedAt             time.Time      `json:"updatedAt"`
}

func (K8sCluster) TableName() string {
	return "k8s_cluster"
}

type K8sDiagnosisTask struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	ClusterID       uint64         `json:"clusterId"`
	ClusterCode     string         `gorm:"size:120" json:"clusterCode"`
	DiagnosisType   string         `gorm:"size:50;not null" json:"diagnosisType"`
	Namespace       string         `gorm:"size:120" json:"namespace"`
	ResourceKind    string         `gorm:"size:80" json:"resourceKind"`
	ResourceName    string         `gorm:"size:255" json:"resourceName"`
	ContainerName   string         `gorm:"size:255" json:"containerName"`
	Question        string         `gorm:"type:text" json:"question"`
	Status          string         `gorm:"size:50;default:pending" json:"status"`
	ErrorMessage    string         `gorm:"type:text" json:"errorMessage"`
	Context         datatypes.JSON `gorm:"type:jsonb" json:"context"`
	Result          datatypes.JSON `gorm:"type:jsonb" json:"result"`
	RetrievedChunks datatypes.JSON `gorm:"type:jsonb" json:"retrievedChunks"`
	CreatedBy       string         `gorm:"size:100" json:"createdBy"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

func (K8sDiagnosisTask) TableName() string {
	return "k8s_diagnosis_task"
}
