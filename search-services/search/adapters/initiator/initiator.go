package initiator

import (
	"context"
	"log/slog"
	"time"

	"yadro.com/course/search/core"
)

type Initiator struct {
	log *slog.Logger
	ttl time.Duration
	svc core.Initiator
}

func NewInitiator(ttl time.Duration, svc core.Initiator, log *slog.Logger) *Initiator {
	return &Initiator{
		ttl: ttl,
		svc: svc,
		log: log,
	}
}

func (i *Initiator) Start(ctx context.Context) {
	go i.runOnce(ctx)

	ticker := time.NewTicker(i.ttl)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				i.runOnce(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (i *Initiator) runOnce(ctx context.Context) {
	if err := i.svc.BuildIndex(ctx); err != nil {
		i.log.Error("build index failed", "error", err)
	}
}
