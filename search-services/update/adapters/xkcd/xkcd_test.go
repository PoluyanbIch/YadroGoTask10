package xkcd

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	log := slog.Default()

	c, err := NewClient(srv.URL, time.Second, log)
	require.NoError(t, err)

	return c
}

func TestClientGet_OK(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/1/info.0.json", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"num": 1,
			"img": "img.png",
			"title": "Title",
			"transcript": "Desc",
			"alt": "Alt"
		}`))
	})
	info, err := client.Get(context.Background(), 1)
	require.NoError(t, err)

	require.Equal(t, 1, info.ID)
	require.Equal(t, "img.png", info.URL)
	require.Equal(t, "Title", info.Title)
	require.Equal(t, "Desc", info.Description)
	require.Equal(t, "Alt", info.Alt)
}

func TestClientGet_UsesSafeTitle(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/1/info.0.json", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"num": 1,
			"img": "img.png",
			"title": "",
			"safe_title": "Safe",
			"transcript": "Desc",
			"alt": "Alt"
		}`))
	})

	info, err := client.Get(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "Safe", info.Title)
}

func TestClientGet_HTTPError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.Get(context.Background(), 1)
	require.Error(t, err)
}

func TestClientGet_InvalidJSON(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{`))
	})

	_, err := client.Get(context.Background(), 1)
	require.Error(t, err)
}

func TestClientLastID_OK(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/info.0.json", r.URL.Path)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"num": 42}`))
	})

	id, err := client.LastID(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, id)
}
