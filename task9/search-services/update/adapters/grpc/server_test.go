package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

type mockUpdater struct {
	updateFn func(ctx context.Context) error
	statsFn  func(ctx context.Context) (core.ServiceStats, error)
	statusFn func(ctx context.Context) core.ServiceStatus
	dropFn   func(ctx context.Context) error
}

func (m *mockUpdater) Update(ctx context.Context) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx)
}

func (m *mockUpdater) Stats(ctx context.Context) (core.ServiceStats, error) {
	if m.statsFn == nil {
		return core.ServiceStats{}, nil
	}
	return m.statsFn(ctx)
}

func (m *mockUpdater) Status(ctx context.Context) core.ServiceStatus {
	if m.statusFn == nil {
		return core.StatusIdle
	}
	return m.statusFn(ctx)
}

func (m *mockUpdater) Drop(ctx context.Context) error {
	if m.dropFn == nil {
		return nil
	}
	return m.dropFn(ctx)
}

func TestServer_Status_Idle(t *testing.T) {
	s := NewServer(&mockUpdater{
		statusFn: func(ctx context.Context) core.ServiceStatus {
			return core.StatusIdle
		},
	})

	resp, err := s.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, updatepb.Status_STATUS_IDLE, resp.Status)
}

func TestServer_Status_Running(t *testing.T) {
	s := NewServer(&mockUpdater{
		statusFn: func(ctx context.Context) core.ServiceStatus {
			return core.StatusRunning
		},
	})

	resp, err := s.Status(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, updatepb.Status_STATUS_RUNNING, resp.Status)
}

func TestServer_Update_Success(t *testing.T) {
	s := NewServer(&mockUpdater{
		updateFn: func(ctx context.Context) error {
			return nil
		},
	})

	resp, err := s.Update(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestServer_Update_AlreadyExists(t *testing.T) {
	s := NewServer(&mockUpdater{
		updateFn: func(ctx context.Context) error {
			return core.ErrAlreadyExists
		},
	})

	resp, err := s.Update(context.Background(), &emptypb.Empty{})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, core.ErrAlreadyExists.Error(), st.Message())
}

func TestServer_Update_InternalError(t *testing.T) {
	expErr := assert.AnError

	s := NewServer(&mockUpdater{
		updateFn: func(ctx context.Context) error {
			return expErr
		},
	})

	resp, err := s.Update(context.Background(), &emptypb.Empty{})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, expErr.Error(), st.Message())
}

func TestServer_Stats_InternalError(t *testing.T) {
	expErr := assert.AnError

	s := NewServer(&mockUpdater{
		statsFn: func(ctx context.Context) (core.ServiceStats, error) {
			return core.ServiceStats{}, expErr
		},
	})

	resp, err := s.Stats(context.Background(), &emptypb.Empty{})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, expErr.Error(), st.Message())
}
