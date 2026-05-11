package usecase

import (
	"context"
	"document-convert-service-new/internal/camunda"
	"document-convert-service-new/internal/converter"
	"document-convert-service-new/internal/model"
	"document-convert-service-new/internal/storage/idempotency"
	"document-convert-service-new/internal/storage/postgres"
	"document-convert-service-new/internal/storage/s3"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"
)

type UseCase struct {
	redis    *idempotency.RedisClient
	camunda  *camunda.CamundaClient
	postgres *postgres.PostgresClient
	s3       *s3.S3Client
}

func NewUseCase(redis *idempotency.RedisClient, camunda *camunda.CamundaClient, postgres *postgres.PostgresClient, s3 *s3.S3Client) *UseCase {
	return &UseCase{
		redis:    redis,
		camunda:  camunda,
		postgres: postgres,
		s3:       s3,
	}
}

func pdfKeyFrom(htmlKey string) string {
	name := strings.TrimSuffix(path.Base(htmlKey), ".html")
	return "pdf/" + name + ".pdf"
}

func (u *UseCase) Process(ctx context.Context, data []byte) error {
	var req model.ConvertRequest

	if err := json.Unmarshal(data, &req); err != nil {
		slog.Error("unmarshal request", "error", err)
		return fmt.Errorf("unmarshal request: %w", err)
	}

	acquired, err := u.redis.TryAcquire(ctx, req.RequestID)
	if err != nil {
		slog.Error("try acquire idempotency key", "error", err, "request_id", req.RequestID)
		return fmt.Errorf("try acquire: %w", err)
	}
	if !acquired {
		slog.Info("skipping duplicate", "request_id", req.RequestID)
		return nil
	}

	if err := u.pipeline(ctx, &req); err != nil {
		slog.Error("pipeline error", "error", err, "request_id", req.RequestID)
		_ = u.redis.DeleteKey(ctx, req.RequestID)
		return fmt.Errorf("pipeline: %w", err)
	}
	return nil
}

func (u *UseCase) pipeline(ctx context.Context, req *model.ConvertRequest) error {
	_ = u.postgres.UpsertStatus(ctx, req.RequestID, "processing", "")

	html, err := u.s3.GetObject(ctx, req.Bucket, req.HtmlS3Key)
	if err != nil {
		_ = u.postgres.UpsertStatus(ctx, req.RequestID, "error", "")
		slog.Error("fetch html from s3", "error", err, "bucket", req.Bucket, "key", req.HtmlS3Key)
		return fmt.Errorf("fetch html from s3: %w", err)
	}

	pdf, err := converter.HTMLToPDF(ctx, html)
	if err != nil {
		_ = u.postgres.UpsertStatus(ctx, req.RequestID, "error", "")
		slog.Error("convert html to pdf", "error", err, "request_id", req.RequestID)
		return fmt.Errorf("convert html to pdf: %w", err)
	}

	pdfKey := pdfKeyFrom(req.HtmlS3Key)
	if err := u.s3.PutObject(ctx, req.Bucket, "application/pdf", pdfKey, pdf); err != nil {
		_ = u.postgres.UpsertStatus(ctx, req.RequestID, "error", "")
		slog.Error("upload pdf to s3", "error", err, "bucket", req.Bucket, "key", pdfKey)
		return fmt.Errorf("upload pdf to s3: %w", err)
	}

	if err := u.postgres.UpsertStatus(ctx, req.RequestID, "completed", pdfKey); err != nil {
		slog.Error("update status in postgres", "error", err, "request_id", req.RequestID)
		return fmt.Errorf("update status in postgres: %w", err)
	}

	if err := u.camunda.SendMessage(ctx, req.CollelationKey, req.RequestID, pdfKey); err != nil {
		slog.Error("send camunda message", "error", err, "request_id", req.RequestID)
		return fmt.Errorf("send camunda message: %w", err)
	}

	slog.Info("processing completed", "request_id", req.RequestID, "pdf_key", pdfKey)

	return nil
}
