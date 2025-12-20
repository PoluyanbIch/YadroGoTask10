package nats

import (
	"context"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type NatsPublisher struct {
	log  *slog.Logger
	conn *nats.Conn
}

func NewNatsPublisher(log *slog.Logger, url string) (*NatsPublisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsPublisher{
		log:  log,
		conn: conn,
	}, nil
}

func (p *NatsPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	if err := p.conn.Publish(subject, data); err != nil {
		p.log.Error("Publish error", "error", err)
		return err
	}
	if err := p.conn.FlushWithContext(ctx); err != nil {
		p.log.Error("Flush error", "error", err)
		return err
	}
	return nil
}
