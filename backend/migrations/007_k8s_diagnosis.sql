CREATE TABLE IF NOT EXISTS k8s_cluster (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    cluster_code VARCHAR(120) NOT NULL UNIQUE,
    api_server TEXT NOT NULL,
    auth_type VARCHAR(50) DEFAULT 'bearer_token',
    credential_ref TEXT,
    allowed_namespaces JSONB,
    insecure_skip_tls_verify BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS k8s_diagnosis_task (
    id BIGSERIAL PRIMARY KEY,
    cluster_id BIGINT,
    cluster_code VARCHAR(120),
    diagnosis_type VARCHAR(50) NOT NULL,
    namespace VARCHAR(120),
    resource_kind VARCHAR(80),
    resource_name VARCHAR(255),
    container_name VARCHAR(255),
    question TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    error_message TEXT,
    context JSONB,
    result JSONB,
    retrieved_chunks JSONB,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_k8s_cluster_code ON k8s_cluster(cluster_code);
CREATE INDEX IF NOT EXISTS idx_k8s_diagnosis_cluster ON k8s_diagnosis_task(cluster_id);
CREATE INDEX IF NOT EXISTS idx_k8s_diagnosis_resource ON k8s_diagnosis_task(namespace, resource_kind, resource_name);
