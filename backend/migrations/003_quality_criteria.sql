CREATE TABLE IF NOT EXISTS quality_criteria (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(120) NOT NULL,
    content TEXT NOT NULL,
    is_default BOOLEAN DEFAULT false,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_quality_criteria_default ON quality_criteria(is_default);
