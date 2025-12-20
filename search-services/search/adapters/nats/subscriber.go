package nats

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type NatsSubscriber struct {
	log  *slog.Logger
	conn *nats.Conn
}

func NewNatsSubscriber(log *slog.Logger, url string) (*NatsSubscriber, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		log.Error("nats connect failed", "error", err, "url", url)
		return nil, err
	}
	return &NatsSubscriber{
		log:  log,
		conn: conn,
	}, nil
}

func (s *NatsSubscriber) Subscribe(ctx context.Context, subject string, handler func(context.Context, []byte) error) error {
	_, err := s.conn.Subscribe(subject, func(msg *nats.Msg) {
		if err := handler(ctx, msg.Data); err != nil {
			s.log.Error("Handler subscribe error", "error", err)
		}
	})
	if err != nil {
		s.log.Error("Nats subscribe error", "error", err, "subject", subject)
		return err
	}
	return nil
}
