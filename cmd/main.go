package main

import (
	"context"
	"document-convert-service-new/config"
	"document-convert-service-new/internal/camunda"
	"document-convert-service-new/internal/consumer"
	"document-convert-service-new/internal/model"
	"document-convert-service-new/internal/pool"
	"document-convert-service-new/internal/storage/idempotency"
	"document-convert-service-new/internal/storage/postgres"
	"document-convert-service-new/internal/storage/s3"
	"document-convert-service-new/internal/usecase"
	"runtime"
	"time"

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

	db, err := postgres.NewPostgresClient(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("init postgres client", "error", err)
		os.Exit(1)
	}
	defer db.Close()

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

	camundaClient := camunda.NewCamundaClient(cfg.CamundaBaseURL, cfg.CamundaMessageName)
	usecase := usecase.NewUseCase(rdb, camundaClient, db, s3Client)

	dispatch := make(chan model.Job, cfg.ChannelBuffer)
	numWorkers := runtime.NumCPU()
	wg := pool.RunWorkerPool(dispatch, usecase, numWorkers)

	slog.Info("starting", "topic", cfg.KafkaTopic, "workers", numWorkers)

	go func() {
		defer close(dispatch)

		if err := consumer.RunConsumerGroup(ctx, cfg.KafkaBrokers, cfg.KafkaGroupID, cfg.KafkaTopic, dispatch); err != nil {
			if ctx.Err() == nil {
				slog.Error("consumer stopped unexpectedly", "err", err)
				cancel()
			}
		}
	}()

	<-ctx.Done()
	slog.Info("signal received, draining workers")

	drainDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(drainDone)
	}()

	select {
	case <-drainDone:
		slog.Info("all workers drained, exiting")
	case <-time.After(30 * time.Second):
		slog.Warn("timeout reached, force exiting")
	}

}
