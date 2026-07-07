CREATE TABLE IF NOT EXISTS app_user (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(120),
    password_hash TEXT NOT NULL,
    role VARCHAR(30) NOT NULL DEFAULT 'user',
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMP,
    password_updated_at TIMESTAMP,
    created_by BIGINT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    CONSTRAINT chk_app_user_role CHECK (role IN ('admin', 'user'))
);

CREATE TABLE IF NOT EXISTS login_audit (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES app_user(id),
    username VARCHAR(100),
    success BOOLEAN NOT NULL,
    failure_reason TEXT,
    client_ip VARCHAR(100),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS conversation (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id),
    title VARCHAR(255),
    conversation_type VARCHAR(50) DEFAULT 'qa',
    status VARCHAR(50) DEFAULT 'active',
    last_message_at TIMESTAMP DEFAULT now(),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS conversation_message (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES app_user(id),
    role VARCHAR(30) NOT NULL,
    content TEXT NOT NULL,
    message_type VARCHAR(50) DEFAULT 'text',
    metadata JSONB,
    created_at TIMESTAMP DEFAULT now(),
    CONSTRAINT chk_conversation_message_role CHECK (role IN ('user', 'assistant', 'system', 'tool'))
);

CREATE TABLE IF NOT EXISTS conversation_summary (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL UNIQUE REFERENCES conversation(id) ON DELETE CASCADE,
    summary TEXT,
    message_count INT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS task_state (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES app_user(id),
    conversation_id BIGINT NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
    task_type VARCHAR(50) NOT NULL,
    task_status VARCHAR(50) DEFAULT 'running',
    state_data JSONB,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tool_call_record (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES app_user(id),
    conversation_id BIGINT REFERENCES conversation(id),
    task_id BIGINT,
    tool_name VARCHAR(120),
    request JSONB,
    response JSONB,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS context_snapshot (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES app_user(id),
    conversation_id BIGINT REFERENCES conversation(id),
    task_id BIGINT,
    snapshot_type VARCHAR(50),
    content JSONB,
    created_at TIMESTAMP DEFAULT now()
);

ALTER TABLE qa_record ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES app_user(id);
ALTER TABLE qa_record ADD COLUMN IF NOT EXISTS conversation_id BIGINT REFERENCES conversation(id);

CREATE INDEX IF NOT EXISTS idx_conversation_user ON conversation(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_message_conversation ON conversation_message(conversation_id);
CREATE INDEX IF NOT EXISTS idx_qa_record_user ON qa_record(user_id);
CREATE INDEX IF NOT EXISTS idx_qa_record_conversation ON qa_record(conversation_id);
