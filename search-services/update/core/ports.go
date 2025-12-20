package core

import (
	"context"
)

//go:generate mockgen -destination=mock_updater.go -package=core . Updater
type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (ServiceStats, error)
	Status(context.Context) ServiceStatus
	Drop(context.Context) error
}

//go:generate mockgen -destination=mock_db_test.go -package=core . DB
type DB interface {
	Add(context.Context, Comics) error
	Stats(context.Context) (DBStats, error)
	Drop(context.Context) error
	IDs(context.Context) ([]int, error)
}

//go:generate mockgen -destination=mock_publisher_test.go -package=core . Publisher
type Publisher interface {
	Publish(context.Context, string, []byte) error
}

//go:generate mockgen -destination=mock_xkcd_test.go -package=core . XKCD
type XKCD interface {
	Get(context.Context, int) (XKCDInfo, error)
	LastID(context.Context) (int, error)
}

//go:generate mockgen -destination=mock_words_test.go -package=core . Words
type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
