package server

import (
	"github.com/rx3lixir/user-service/internal/db"
	userPb "github.com/rx3lixir/user-service/user-grpc/gen/go"
)

type Server struct {
	storer *db.PostgresStore
	userPb.UnimplementedUserServiceServer
}

func NewServer(storer *db.PostgresStore) *Server {
	return &Server{
		storer: storer,
	}
}

func (s *Server) CreateUser(ctx context.Context, u *pb.UserReq) (*pb.UserRes, error) {
	user, err := s.storer.CreateUser(ctx, toStorerUser(u))
	if err != nil {
		return nil, err
	}

	return toPBUserRes(user), nil
}

func (s *Server) GetUser(ctx context.Context, u *pb.UserReq) (*pb.UserRes, error) {
	user, err := s.storer.GetUser(ctx, u.GetEmail())
	if err != nil {
		return nil, err
	}

	return toPBUserRes(user), nil
}

func (s *Server) ListUsers(ctx context.Context, u *pb.UserReq) (*pb.ListUserRes, error) {
	users, err := s.storer.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	lur := make([]*pb.UserRes, 0, len(users))
	for _, user := range users {
		lur = append(lur, toPBUserRes(user))
	}

	return &pb.ListUserRes{
		Users: lur,
	}, nil
}

func (s *Server) UpdateUser(ctx context.Context, u *pb.UserReq) (*pb.UserRes, error) {
	user, err := s.storer.GetUser(ctx, u.GetEmail())
	if err != nil {
		return nil, err
	}

	patchUserReq(user, u)
	ur, err := s.storer.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return toPBUserRes(ur), nil
}

func (s *Server) DeleteUser(ctx context.Context, u *pb.UserReq) (*pb.UserRes, error) {
	err := s.storer.DeleteUser(ctx, u.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.UserRes{}, nil
}
