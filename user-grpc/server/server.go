package server

import (
	"context"

	"github.com/rx3lixir/user-service/internal/db"
	"github.com/rx3lixir/user-service/internal/logger"
	"github.com/rx3lixir/user-service/pkg/password"
	pb "github.com/rx3lixir/user-service/user-grpc/gen/go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	storer db.UserStore
	pb.UnimplementedUserServiceServer
	log logger.Logger
}

func NewServer(storer db.UserStore, log logger.Logger) *Server {
	return &Server{
		storer: storer,
		log:    log,
	}
}

func (s *Server) CreateUser(ctx context.Context, req *pb.UserReq) (*pb.UserRes, error) {
	s.log.Info("starting create user",
		"method", "CreateUser",
		"name", req.GetName(),
		"email", req.GetEmail(),
		"is_admin", req.GetIsAdmin(),
	)

	user := &db.User{
		Name:     req.GetName(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
		IsAdmin:  req.GetIsAdmin(),
	}

	if err := s.storer.CreateUser(ctx, user); err != nil {
		s.log.Error("error creating user", "user", user.Email, "error", err)
		return nil, err
	}

	s.log.Info("user created successfully",
		"method", "CreateUser",
		"user_id", user.Id,
		"name", user.Name,
		"email", user.Email,
	)

	return toPBUserRes(user), nil
}

func (s *Server) GetUser(ctx context.Context, req *pb.UserReq) (*pb.UserRes, error) {
	s.log.Info("starting get user",
		"method", "GetUser",
		"id", req.GetId(),
		"email", req.GetEmail(),
	)

	user := new(db.User)
	var err error

	// Решаем, как искать пользователя - по ID или email
	if req.GetId() > 0 {
		user, err = s.storer.GetUserByID(ctx, int(req.GetId()))
	} else if req.GetEmail() != "" {
		user, err = s.storer.GetUserByEmail(ctx, req.GetEmail())
	} else {
		err := status.Error(codes.InvalidArgument, "id or email required")
		s.log.Error("invalid arguments for get user",
			"method", "GetUser",
			"error", err,
		)
		return nil, err
	}

	if err != nil {
		s.log.Error("failed to get user",
			"method", "GetUser",
			"error", err,
			"id", req.GetId(),
			"email", req.GetEmail(),
		)
		return nil, err
	}

	s.log.Debug("user retrieved successfully",
		"method", "GetUser",
		"user_id", user.Id,
		"name", user.Name,
		"email", user.Email,
	)

	return toPBUserRes(user), nil
}

func (s *Server) ListUsers(ctx context.Context, req *pb.UserReq) (*pb.ListUserRes, error) {
	s.log.Info("starting list users",
		"method", "ListUsers",
	)

	users, err := s.storer.GetUsers(ctx)
	if err != nil {
		s.log.Error("failed to list users",
			"method", "ListUsers",
			"error", err,
		)
		return nil, err
	}

	pbUsers := make([]*pb.UserRes, 0, len(users))

	for _, user := range users {
		pbUsers = append(pbUsers, toPBUserRes(user))
	}

	s.log.Info("users listed successfully",
		"method", "ListUsers",
		"count", len(users),
	)

	return &pb.ListUserRes{
		Users: pbUsers,
	}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req *pb.UserReq) (*pb.UserRes, error) {
	s.log.Info("starting update user",
		"method", "UpdateUser",
		"user_id", req.GetId(),
	)

	// Обработка пустого ID
	if req.GetId() == 0 {
		err := status.Error(codes.InvalidArgument, "user id required")

		s.log.Error("invalid arguments for update user",
			"method", "UpdateUser",
			"error", err,
		)

		return nil, err
	}

	user, err := s.storer.GetUserByID(ctx, int(req.GetId()))
	if err != nil {
		s.log.Error("failed to get user for update",
			"method", "UpdateUser",
			"user_id", req.GetId(),
			"error", err,
		)
		return nil, err
	}

	// Обновляем только заполненные поля
	if req.GetName() != "" {
		user.Name = req.GetName()
	}

	if req.GetEmail() != "" {
		user.Email = req.GetEmail()
	}

	if req.GetPassword() != "" {
		hashedPassword, err := password.Hash(req.GetPassword())
		if err != nil {
			return nil, err
		}
		user.Password = hashedPassword
	}

	user.IsAdmin = req.GetIsAdmin()

	if err := s.storer.UpdateUser(ctx, user); err != nil {
		s.log.Error("failed to update user",
			"method", "UpdateUser",
			"user_id", req.GetId(),
			"error", err,
		)
		return nil, err
	}

	s.log.Info("user updated successfully",
		"method", "UpdateUser",
		"user_id", user.Id,
	)

	return toPBUserRes(user), nil
}

func (s *Server) DeleteUser(ctx context.Context, req *pb.UserReq) (*pb.UserRes, error) {
	s.log.Info("starting delete user",
		"method", "DeleteUser",
		"user_id", req.GetId(),
	)

	if req.GetId() == 0 {
		err := status.Error(codes.InvalidArgument, "user id required")
		s.log.Error("invalid arguments for delete user",
			"method", "DeleteUser",
			"error", err,
		)
		return nil, err
	}

	if err := s.storer.DeleteUser(ctx, int(req.GetId())); err != nil {
		s.log.Error("failed to delete user",
			"method", "DeleteUser",
			"user_id", req.GetId(),
			"error", err,
		)
		return nil, err
	}

	return &pb.UserRes{}, nil
}
