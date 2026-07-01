CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS kb_document (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    system_name VARCHAR(100),
    component_name VARCHAR(100),
    doc_type VARCHAR(100),
    version VARCHAR(50) DEFAULT 'v1.0',
    status VARCHAR(50) DEFAULT 'draft',
    tags TEXT,
    summary TEXT,
    quality_score INT DEFAULT 0,
    quality_result JSONB,
    created_by VARCHAR(100),
    reviewed_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    reviewed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kb_chunk (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES kb_document(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    source_title VARCHAR(255),
    source_section VARCHAR(255),
    source_page INT,
    token_count INT DEFAULT 0,
    search_text TEXT,
    keywords JSONB,
    possible_questions JSONB,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_kb_document_status ON kb_document(status);
CREATE INDEX IF NOT EXISTS idx_kb_document_filters ON kb_document(system_name, component_name, doc_type);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_document_id ON kb_chunk(document_id);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_content_trgm ON kb_chunk USING gin (content gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_search_text_trgm ON kb_chunk USING gin (search_text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_source_section_trgm ON kb_chunk USING gin (source_section gin_trgm_ops);

CREATE TABLE IF NOT EXISTS qa_record (
    id BIGSERIAL PRIMARY KEY,
    question TEXT NOT NULL,
    answer TEXT NOT NULL,
    retrieved_chunks JSONB,
    model_name VARCHAR(100),
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS kb_review_record (
    id BIGSERIAL PRIMARY KEY,
    document_id BIGINT NOT NULL REFERENCES kb_document(id) ON DELETE CASCADE,
    from_status VARCHAR(50),
    to_status VARCHAR(50),
    reviewer VARCHAR(100),
    comment TEXT,
    created_at TIMESTAMP DEFAULT now()
);
