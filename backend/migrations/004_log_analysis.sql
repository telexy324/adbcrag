CREATE TABLE IF NOT EXISTS log_source (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    source_type VARCHAR(50) NOT NULL,
    system_name VARCHAR(100),
    component_name VARCHAR(100),
    environment VARCHAR(50),
    endpoint TEXT,
    username VARCHAR(255),
    credential_ref TEXT,
    es_index_pattern VARCHAR(255),
    es_time_field VARCHAR(100),
    server_host VARCHAR(255),
    server_port INT DEFAULT 22,
    auth_type VARCHAR(50),
    log_path TEXT,
    path_allowlist JSONB,
    enabled BOOLEAN DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_log_source_type ON log_source(source_type);
CREATE INDEX IF NOT EXISTS idx_log_source_filters ON log_source(system_name, component_name, environment);

CREATE TABLE IF NOT EXISTS log_analysis_task (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT REFERENCES log_source(id),
    question TEXT,
    system_name VARCHAR(100),
    component_name VARCHAR(100),
    time_start TIMESTAMP,
    time_end TIMESTAMP,
    keyword TEXT,
    log_level VARCHAR(50),
    status VARCHAR(50) DEFAULT 'pending',
    sample_count INT DEFAULT 0,
    error_message TEXT,
    result JSONB,
    retrieved_chunks JSONB,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_log_analysis_source_id ON log_analysis_task(source_id);
CREATE INDEX IF NOT EXISTS idx_log_analysis_created_at ON log_analysis_task(created_at);
