CREATE TABLE IF NOT EXISTS llm_config (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    base_url TEXT NOT NULL,
    model VARCHAR(120) NOT NULL,
    api_key_ref TEXT,
    temperature DOUBLE PRECISION DEFAULT 0.2,
    is_default BOOLEAN DEFAULT false,
    enabled BOOLEAN DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_llm_config_default ON llm_config(is_default);
CREATE INDEX IF NOT EXISTS idx_llm_config_provider ON llm_config(provider);
