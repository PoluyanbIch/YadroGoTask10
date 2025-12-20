package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

func TestServer_Search_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	searcher := core.NewMockSearcher(ctrl)
	server := NewServer(searcher)

	req := &searchpb.SearchRequest{
		Phrase: "test",
		Limit:  10,
	}

	searcher.
		EXPECT().
		Search(gomock.Any(), "test", 10).
		Return([]core.Comic{
			{ID: 1, URL: "url1"},
			{ID: 2, URL: "url2"},
		}, nil)

	resp, err := server.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Total != 2 {
		t.Fatalf("expected total=2, got %d", resp.Total)
	}

	if resp.Comics[0].Id != 1 || resp.Comics[1].Id != 2 {
		t.Fatal("invalid comics mapping")
	}
}

func TestServer_Search_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	searcher := core.NewMockSearcher(ctrl)
	server := NewServer(searcher)

	searcher.
		EXPECT().
		Search(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("boom"))

	_, err := server.Search(context.Background(), &searchpb.SearchRequest{})
	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected grpc status error")
	}

	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", st.Code())
	}
}

func TestServer_Ping_ServiceNil(t *testing.T) {
	server := NewServer(nil)

	_, err := server.Ping(context.Background(), &emptypb.Empty{})
	if err == nil {
		t.Fatal("expected error")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.Internal {
		t.Fatalf("expected Internal, got %v", st.Code())
	}
}

func TestServer_ISearch_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	searcher := core.NewMockSearcher(ctrl)
	server := NewServer(searcher)

	searcher.EXPECT().
		ISearch(gomock.Any(), "dog", 1).
		Return([]core.Comic{
			{ID: 42, URL: "url"},
		}, nil)

	resp, err := server.ISearch(context.Background(), &searchpb.SearchRequest{
		Phrase: "dog",
		Limit:  1,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), resp.Total)
}
