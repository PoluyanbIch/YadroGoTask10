package core

import (
	"context"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestService(t *testing.T, db DB, xkcd XKCD, words Words, publisher Publisher) *Service {
	t.Helper()
	log := slog.Default()
	s, err := NewService(log, db, xkcd, words, publisher, 2)
	require.NoError(t, err)
	return s
}

func TestNewServiceWrongConcurrency(t *testing.T) {
	s, err := NewService(nil, nil, nil, nil, nil, 0)
	require.Error(t, err)
	require.Nil(t, s)
}

func TestServiceStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	s := newTestService(
		t,
		NewMockDB(ctrl),
		NewMockXKCD(ctrl),
		NewMockWords(ctrl),
		NewMockPublisher(ctrl),
	)
	require.Equal(t, StatusIdle, s.Status(context.Background()))
}

func TestServiceStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	db := NewMockDB(ctrl)
	xkcd := NewMockXKCD(ctrl)

	db.EXPECT().Stats(gomock.Any()).Return(DBStats{WordsTotal: 10}, nil)
	xkcd.EXPECT().LastID(gomock.Any()).Return(101, nil)

	s := newTestService(t, db, xkcd, nil, nil)
	stats, err := s.Stats(context.Background())
	require.NoError(t, err)

	require.Equal(t, 100, stats.ComicsTotal)
	require.Equal(t, 10, stats.WordsTotal)
}

func TestServiceDrop(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	db := NewMockDB(ctrl)
	pub := NewMockPublisher(ctrl)

	db.EXPECT().Drop(gomock.Any()).Return(nil)
	pub.EXPECT().Publish(gomock.Any(), "xkcd.db.drop", gomock.Any()).Return(nil)

	s := newTestService(t, db, nil, nil, pub)
	require.NoError(t, s.Drop(context.Background()))
}

func TestServiceUpdateOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	db := NewMockDB(ctrl)
	xkcd := NewMockXKCD(ctrl)
	words := NewMockWords(ctrl)
	pub := NewMockPublisher(ctrl)

	db.EXPECT().IDs(gomock.Any()).Return([]int{}, nil)
	xkcd.EXPECT().LastID(gomock.Any()).Return(1, nil)
	xkcd.EXPECT().Get(gomock.Any(), 1).Return(XKCDInfo{
		ID:          1,
		URL:         "url",
		Title:       "title",
		Description: "desc",
		Alt:         "alt",
	}, nil)

	words.EXPECT().Norm(gomock.Any(), gomock.Any()).Return([]string{"a", "b"}, nil).Times(3)
	db.EXPECT().Add(gomock.Any(), gomock.Any()).Return(nil)
	pub.EXPECT().Publish(gomock.Any(), "xkcd.db.update", gomock.Any()).Return(nil)

	s := newTestService(t, db, xkcd, words, pub)
	require.NoError(t, s.Update(context.Background()))
}

func TestServiceUpdateAlreadyRunning(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	s := newTestService(
		t,
		NewMockDB(ctrl),
		NewMockXKCD(ctrl),
		NewMockWords(ctrl),
		NewMockPublisher(ctrl),
	)

	atomic.StoreInt32(&s.isUpdating, 1)
	err := s.Update(context.Background())
	require.ErrorIs(t, err, ErrUpdateInProgress)
}
