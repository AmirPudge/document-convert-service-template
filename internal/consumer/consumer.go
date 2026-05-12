package consumer

import (
	"context"
	"document-convert-service-new/internal/model"
	"log/slog"

	"github.com/IBM/sarama"
)

type Handler struct {
	dispatch chan<- model.Job
}

func (h *Handler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *Handler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *Handler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			data := make([]byte, len(msg.Value))
			copy(data, msg.Value)

			job := model.Job{
				Data: data,
				Ack:  func() { session.MarkMessage(msg, "") },
			}

			select {
			case h.dispatch <- job:
			case <-session.Context().Done():
				return nil
			}
		case <-session.Context().Done():
			return nil
		}
	}

}

func RunConsumerGroup(ctx context.Context, brokers []string, groupID, topic string, dispatch chan<- model.Job) error {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_6_0_0
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return err
	}
	defer group.Close()

	go func() {
		for err := range group.Errors() {
			slog.Error("Kafka consumer group error", "error", err)
		}
	}()

	h := &Handler{dispatch: dispatch}

	for {
		if err := group.Consume(ctx, []string{topic}, h); err != nil {
			slog.Error("Kafka consumer group error", "error", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

	}
}
