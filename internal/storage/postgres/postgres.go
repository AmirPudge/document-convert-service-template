package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresClient struct {
	pool *pgxpool.Pool
}

func NewPostgresClient(ctx context.Context, dsn string) (*PostgresClient, error) {
	pool, err := pgxpool.New(ctx, dsn)

	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	return &PostgresClient{pool: pool}, nil
}

func (p *PostgresClient) UpsertStatus(ctx context.Context, requestID, status, pdfS3Key string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO document_statuses (request_id, status, pdf_s3_key, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (request_id) DO UPDATE SET
			status 	   = EXCLUDED.status,
			pdf_s3_key = EXCLUDED.pdf_s3_key,
			updated_at = NOW()
	`, requestID, status, pdfS3Key)
	if err != nil {
		return fmt.Errorf("upsert doc_status: %w", err)
	}
	return nil
}

func (p *PostgresClient) Close() {
	p.pool.Close()
}
