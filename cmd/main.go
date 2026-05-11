package main

import (
	"context"
	"document-convert-service-new/config"
	"document-convert-service-new/internal/camunda"
	"document-convert-service-new/internal/converter"
	"document-convert-service-new/internal/storage/idempotency"
	"document-convert-service-new/internal/storage/s3"

	"log/slog"
	"os"
	"os/signal"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	s3Client, err := s3.NewS3Client(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Region, cfg.S3UsePathStyle)
	if err != nil {
		slog.Error("init s3 client", "error", err)
		os.Exit(1)
	}

	rdb, err := idempotency.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		slog.Error("init redis client", "error", err)
		os.Exit(1)
	}

	defer rdb.Close()

	data := []byte("Hello, S3!")
	key := "test/hello.txt"
	if err := s3Client.PutObject(ctx, cfg.S3Bucket, "text/plain", key, data); err != nil {
		slog.Error("put object to s3", "error", err)
		os.Exit(1)
	}

	got, err := s3Client.GetObject(ctx, cfg.S3Bucket, key)
	if err != nil {
		slog.Error("get object from s3", "error", err)
		os.Exit(1)
	}

	slog.Info("s3 test ok", "got", string(got))

	redisData, err := rdb.TryAcquire(ctx, "tes111")
	if err != nil {
		slog.Error("check key in redis", "error", err)
		os.Exit(1)
	} else {
		slog.Info("idempotency key found", "data", redisData)
	}

	htmlData := []byte(`<!DOCTYPE html><html><head><meta charset="utf-8"><style>body { font-family: sans-serif; } .box { color: red; font-size: 24px; }</style></head><body><div class="box">Привет, мир! Тест кириллицы.</div></body></html>`)

	res, err := converter.HTMLToPDF(ctx, htmlData)
	if err != nil {
		slog.Error("convert html to pdf", "error", err)
		os.Exit(1)
	}

	if err := os.WriteFile("output.pdf", res, 0644); err != nil {
		slog.Error("save pdf to file", "error", err)
		os.Exit(1)
	}
	slog.Info("pdf saved to output.pdf")

	slog.Info("html to pdf conversion successful", "pdfSize", len(res))

	slog.Info("redis test ok", "data", redisData)

	camundaClient := camunda.NewCamundaClient(cfg.CamundaBaseURL, cfg.CamundaMessageName)
	if err := camundaClient.SendMessage(ctx, "test-correlation-key", "test-request-id", "test-pdf-s3-key"); err != nil {
		slog.Error("send message to camunda", "error", err)
		os.Exit(1)
	}
}
