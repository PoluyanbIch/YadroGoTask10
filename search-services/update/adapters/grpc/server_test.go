package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

func TestServer_Ping_ServiceNil(t *testing.T) {
	s := NewServer(nil)

	_, err := s.Ping(context.Background(), &emptypb.Empty{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, st.Code())
}

func TestServer_Status_Idle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Status(gomock.Any()).
		Return(core.StatusIdle)

	s := NewServer(updater)

	resp, err := s.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_IDLE, resp.Status)
}

func TestServer_Status_Running(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Status(gomock.Any()).
		Return(core.StatusRunning)

	s := NewServer(updater)

	resp, err := s.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Equal(t, updatepb.Status_STATUS_RUNNING, resp.Status)
}

func TestServer_Update_InProgress(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Update(gomock.Any()).
		Return(core.ErrUpdateInProgress)

	s := NewServer(updater)

	_, err := s.Update(context.Background(), &emptypb.Empty{})
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.AlreadyExists, st.Code())
}

func TestServer_Update_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Update(gomock.Any()).
		Return(errors.New("boom"))

	s := NewServer(updater)

	_, err := s.Update(context.Background(), &emptypb.Empty{})
	require.Error(t, err)

	st, _ := status.FromError(err)
	require.Equal(t, codes.Internal, st.Code())
}

func TestServer_Stats_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	stats := core.ServiceStats{
		DBStats: core.DBStats{
			WordsTotal:    10,
			WordsUnique:   5,
			ComicsFetched: 3,
		},
		ComicsTotal: 100,
	}

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Stats(gomock.Any()).
		Return(stats, nil)

	s := NewServer(updater)

	resp, err := s.Stats(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)

	require.Equal(t, int64(10), resp.WordsTotal)
	require.Equal(t, int64(5), resp.WordsUnique)
	require.Equal(t, int64(3), resp.ComicsFetched)
	require.Equal(t, int64(100), resp.ComicsTotal)
}

func TestServer_Drop_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := core.NewMockUpdater(ctrl)
	updater.EXPECT().
		Drop(gomock.Any()).
		Return(errors.New("fail"))

	s := NewServer(updater)

	_, err := s.Drop(context.Background(), &emptypb.Empty{})
	require.Error(t, err)

	st, _ := status.FromError(err)
	require.Equal(t, codes.Internal, st.Code())
}
