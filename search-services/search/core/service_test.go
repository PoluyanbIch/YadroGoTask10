package core

import (
	context "context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

func newTestService(t *testing.T, db DB, words Words) *Service {
	t.Helper()
	log := slog.Default()
	return NewService(log, db, words)
}

func sampleComics() []DBComic {
	return []DBComic{
		{
			ID:  1,
			URL: "url1",
			Title: map[string]int{
				"hello": 1,
			},
			Description: map[string]int{
				"world": 2,
			},
		},
		{
			ID:  2,
			URL: "url2",
			Title: map[string]int{
				"hello": 1,
			},
			Description: map[string]int{
				"golang": 1,
			},
		},
	}
}

func TestService_Search_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDB(ctrl)
	words := NewMockWords(ctrl)

	ctx := context.Background()

	db.EXPECT().
		Read(gomock.Any()).
		Return(sampleComics(), nil)

	words.EXPECT().
		Norm(gomock.Any(), "hello").
		Return([]string{"hello"}, nil)

	svc := newTestService(t, db, words)

	res, err := svc.Search(ctx, "hello", 10)

	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, 1, res[0].ID)
}

func TestService_Search_WordsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDB(ctrl)
	words := NewMockWords(ctrl)

	words.EXPECT().
		Norm(gomock.Any(), gomock.Any()).
		Return(nil, assert.AnError)

	svc := newTestService(t, db, words)

	_, err := svc.Search(context.Background(), "test", 5)
	require.Error(t, err)
}

func TestService_Search_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDB(ctrl)
	words := NewMockWords(ctrl)

	words.EXPECT().
		Norm(gomock.Any(), gomock.Any()).
		Return([]string{"test"}, nil)

	db.EXPECT().
		Read(gomock.Any()).
		Return(nil, assert.AnError)

	svc := newTestService(t, db, words)

	_, err := svc.Search(context.Background(), "test", 5)
	require.Error(t, err)
}

func TestService_BuildIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDB(ctrl)
	words := NewMockWords(ctrl)

	db.EXPECT().
		Read(gomock.Any()).
		Return(sampleComics(), nil)

	svc := newTestService(t, db, words)
	err := svc.BuildIndex(context.Background())
	require.NoError(t, err)

	require.NotNil(t, svc.index)
	require.NotNil(t, svc.comicsMap)
	require.Contains(t, svc.index, "hello")
}

func TestService_ISearch_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := NewMockDB(ctrl)
	words := NewMockWords(ctrl)

	db.EXPECT().
		Read(gomock.Any()).
		Return(sampleComics(), nil)

	words.EXPECT().
		Norm(gomock.Any(), "hello").
		Return([]string{"hello"}, nil)

	svc := newTestService(t, db, words)
	require.NoError(t, svc.BuildIndex(context.Background()))

	res, err := svc.ISearch(context.Background(), "hello", 5)

	require.NoError(t, err)
	require.NotEmpty(t, res)
}

func TestCalculateTF(t *testing.T) {
	field := map[string]int{
		"hello": 2,
		"world": 1,
	}

	tf := calculateTF("hello", field)
	require.Greater(t, tf, 0.0)
}

func TestCountSubstringInField(t *testing.T) {
	field := map[string]int{
		"hello": 2,
	}

	n := countSubstringInField("hel", field)
	require.Equal(t, 2, n)
}
