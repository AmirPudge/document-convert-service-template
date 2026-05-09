package main

import (
	"context"
	"document-convert-service-new/config"
	"document-convert-service-new/storage/s3"
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
}
