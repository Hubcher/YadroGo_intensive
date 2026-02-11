package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type mockSearcher struct {
	searchFn      func(ctx context.Context, phrase string, limit int) ([]core.Comic, error)
	indexSearchFn func(ctx context.Context, phrase string, limit int) ([]core.Comic, error)
}

func (m *mockSearcher) Search(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
	if m.searchFn == nil {
		return nil, nil
	}
	return m.searchFn(ctx, phrase, limit)
}

func (m *mockSearcher) IndexSearch(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
	if m.indexSearchFn == nil {
		return nil, nil
	}
	return m.indexSearchFn(ctx, phrase, limit)
}

func TestServer_Search_ErrBadArguments(t *testing.T) {
	ms := &mockSearcher{
		searchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
			return nil, core.ErrBadArguments
		},
	}
	s := NewServer(ms)

	resp, err := s.Search(context.Background(), &searchpb.SearchRequest{
		Phrase: "",
		Limit:  0,
	})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, core.ErrBadArguments.Error(), st.Message())
}

func TestServer_Search_InternalError(t *testing.T) {
	expErr := assert.AnError

	ms := &mockSearcher{
		searchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
			return nil, expErr
		},
	}
	s := NewServer(ms)

	resp, err := s.Search(context.Background(), &searchpb.SearchRequest{
		Phrase: "anything",
		Limit:  10,
	})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, expErr.Error(), st.Message())
}

func TestServer_IndexSearch_ErrBadArguments(t *testing.T) {
	ms := &mockSearcher{
		indexSearchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
			return nil, core.ErrBadArguments
		},
	}
	s := NewServer(ms)

	resp, err := s.IndexSearch(context.Background(), &searchpb.SearchRequest{
		Phrase: "",
		Limit:  0,
	})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, core.ErrBadArguments.Error(), st.Message())
}

func TestServer_IndexSearch_InternalError(t *testing.T) {
	expErr := assert.AnError

	ms := &mockSearcher{
		indexSearchFn: func(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
			return nil, expErr
		},
	}
	s := NewServer(ms)

	resp, err := s.IndexSearch(context.Background(), &searchpb.SearchRequest{
		Phrase: "idx",
		Limit:  5,
	})
	assert.Nil(t, resp)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, expErr.Error(), st.Message())
}
