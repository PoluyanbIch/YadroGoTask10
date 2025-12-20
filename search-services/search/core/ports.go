package core

import (
	"context"
)

//go:generate mockgen -destination=mock_searcher.go -package=core . Searcher
type Searcher interface {
	Search(context.Context, string, int) ([]Comic, error)
	ISearch(context.Context, string, int) ([]Comic, error)
}

//go:generate mockgen -destination=mock_db_test.go -package=core . DB
type DB interface {
	Read(context.Context) ([]DBComic, error)
}

//go:generate mockgen -destination=mock_words_test.go -package=core . Words
type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

//go:generate mockgen -destination=mock_subscriber_test.go -package=core . Subscriber
type Subscriber interface {
	Subscribe(context.Context, string, func(context.Context, []byte) error) error
}

//go:generate mockgen -destination=mock_initiator_test.go -package=core . Initiator
type Initiator interface {
	BuildIndex(context.Context) error
}
