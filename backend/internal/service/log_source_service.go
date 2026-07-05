package service

import (
	"context"
	"encoding/json"
	"fmt"
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

type LogCredentials struct {
	Password             string `json:"password,omitempty"`
	PrivateKey           string `json:"privateKey,omitempty"`
	PrivateKeyPassphrase string `json:"privateKeyPassphrase,omitempty"`
}

type LogSourceService struct {
	cfg    *config.Config
	repo   *repository.LogSourceRepository
	crypto *security.CredentialCrypto
	es     client.ElasticsearchClient
	ssh    client.SSHLogClient
}

func NewLogSourceService(cfg *config.Config, repo *repository.LogSourceRepository, crypto *security.CredentialCrypto, es client.ElasticsearchClient, ssh client.SSHLogClient) *LogSourceService {
	return &LogSourceService{cfg: cfg, repo: repo, crypto: crypto, es: es, ssh: ssh}
}

func (s *LogSourceService) List(ctx context.Context) ([]model.LogSource, error) {
	return s.repo.List(ctx)
}

func (s *LogSourceService) Create(ctx context.Context, req dto.SaveLogSourceRequest, createdBy string) (*model.LogSource, error) {
	source, err := s.buildSource(req, nil)
	if err != nil {
		return nil, err
	}
	source.CreatedBy = createdBy
	if err := s.repo.Create(ctx, source); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *LogSourceService) Update(ctx context.Context, id uint64, req dto.SaveLogSourceRequest) (*model.LogSource, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	next, err := s.buildSource(req, existing)
	if err != nil {
		return nil, err
	}
	next.ID = existing.ID
	next.CreatedBy = existing.CreatedBy
	next.CreatedAt = existing.CreatedAt
	if err := s.repo.Update(ctx, next); err != nil {
		return nil, err
	}
	return next, nil
}

func (s *LogSourceService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func (s *LogSourceService) GetWithCredentials(ctx context.Context, id uint64) (*model.LogSource, LogCredentials, error) {
	source, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, LogCredentials{}, err
	}
	credentials, err := s.decryptCredentials(source.CredentialRef)
	if err != nil {
		return nil, LogCredentials{}, err
	}
	return source, credentials, nil
}

func (s *LogSourceService) Test(ctx context.Context, id uint64) error {
	source, credentials, err := s.GetWithCredentials(ctx, id)
	if err != nil {
		return err
	}
	timeout := time.Duration(s.cfg.ESTimeoutSec) * time.Second
	if source.SourceType == model.LogSourceTypeElasticsearch {
		return s.es.Test(ctx, client.ESConfig{Endpoint: source.Endpoint, Username: source.Username, Password: credentials.Password, Timeout: timeout})
	}
	return s.ssh.Test(ctx, client.SSHConfig{
		Host: source.ServerHost, Port: source.ServerPort, Username: source.Username, Password: credentials.Password,
		PrivateKey: credentials.PrivateKey, Passphrase: credentials.PrivateKeyPassphrase, AuthType: source.AuthType,
		Timeout: time.Duration(s.cfg.SSHTimeoutSec) * time.Second,
	})
}

func (s *LogSourceService) buildSource(req dto.SaveLogSourceRequest, existing *model.LogSource) (*model.LogSource, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.SourceType = strings.TrimSpace(req.SourceType)
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.SourceType != model.LogSourceTypeElasticsearch && req.SourceType != model.LogSourceTypeServerFile {
		return nil, fmt.Errorf("unsupported sourceType: %s", req.SourceType)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else if existing != nil {
		enabled = existing.Enabled
	}
	allowlist, _ := json.Marshal(req.PathAllowlist)
	source := &model.LogSource{
		Name: req.Name, SourceType: req.SourceType, SystemName: req.SystemName, ComponentName: req.ComponentName,
		Environment: req.Environment, Endpoint: strings.TrimRight(req.Endpoint, "/"), Username: req.Username,
		ESIndexPattern: req.ESIndexPattern, ESTimeField: req.ESTimeField, ServerHost: req.ServerHost,
		ServerPort: req.ServerPort, AuthType: req.AuthType, LogPath: req.LogPath, PathAllowlist: datatypes.JSON(allowlist),
		Enabled: enabled,
	}
	if source.ServerPort == 0 {
		source.ServerPort = 22
	}
	if source.ESTimeField == "" {
		source.ESTimeField = "@timestamp"
	}
	if source.SourceType == model.LogSourceTypeElasticsearch && (source.Endpoint == "" || source.ESIndexPattern == "") {
		return nil, fmt.Errorf("endpoint and esIndexPattern are required")
	}
	if source.SourceType == model.LogSourceTypeServerFile {
		if source.ServerHost == "" || source.LogPath == "" || len(req.PathAllowlist) == 0 {
			return nil, fmt.Errorf("serverHost, logPath and pathAllowlist are required")
		}
		if source.AuthType == "" {
			source.AuthType = model.LogAuthTypePassword
		}
		if source.AuthType != model.LogAuthTypePassword && source.AuthType != model.LogAuthTypePrivateKey {
			return nil, fmt.Errorf("unsupported authType: %s", source.AuthType)
		}
	}
	credentials := LogCredentials{Password: req.Password, PrivateKey: req.PrivateKey, PrivateKeyPassphrase: req.PrivateKeyPassphrase}
	if existing != nil && credentials.Password == "" && credentials.PrivateKey == "" && credentials.PrivateKeyPassphrase == "" {
		source.CredentialRef = existing.CredentialRef
		return source, nil
	}
	credentialRef, err := s.encryptCredentials(credentials)
	if err != nil {
		return nil, err
	}
	source.CredentialRef = credentialRef
	return source, nil
}

func (s *LogSourceService) encryptCredentials(credentials LogCredentials) (string, error) {
	data, _ := json.Marshal(credentials)
	return s.crypto.Encrypt(string(data))
}

func (s *LogSourceService) decryptCredentials(ref string) (LogCredentials, error) {
	raw, err := s.crypto.Decrypt(ref)
	if err != nil {
		return LogCredentials{}, err
	}
	var credentials LogCredentials
	if raw != "" {
		err = json.Unmarshal([]byte(raw), &credentials)
	}
	return credentials, err
}
