CREATE TABLE IF NOT EXISTS document_statuses (
    request_id  TEXT        PRIMARY KEY,
    status      TEXT        NOT NULL,
    pdf_s3_key  TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
