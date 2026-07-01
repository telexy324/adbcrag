CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE kb_chunk
    ADD COLUMN IF NOT EXISTS search_text TEXT,
    ADD COLUMN IF NOT EXISTS keywords JSONB,
    ADD COLUMN IF NOT EXISTS possible_questions JSONB;

CREATE INDEX IF NOT EXISTS idx_kb_chunk_content_trgm ON kb_chunk USING gin (content gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_search_text_trgm ON kb_chunk USING gin (search_text gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_kb_chunk_source_section_trgm ON kb_chunk USING gin (source_section gin_trgm_ops);
