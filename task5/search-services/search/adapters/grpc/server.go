package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Search(ctx context.Context, req *searchpb.SearchRequest) (*searchpb.SearchReply, error) {

	comics, err := s.service.Search(ctx, req.GetPhrase(), int(req.GetLimit()))

	if err != nil {
		if errors.Is(err, core.ErrBadArguments) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &searchpb.SearchReply{
		Comics: make([]*searchpb.Comic, 0, len(comics)),
	}
	for _, c := range comics {
		resp.Comics = append(resp.Comics, &searchpb.Comic{
			Id:  int64(c.ID),
			Url: c.URL,
		})
	}
	return resp, nil

}
