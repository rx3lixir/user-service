package server

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rx3lixir/user-service/internal/db"
	pb "github.com/rx3lixir/user-service/user-grpc/gen/go"
)

// Преобразует объект User из базы данных в протобаф-объект UserRes
func toPBUserRes(u *db.User) *pb.UserRes {
	return &pb.UserRes{
		Id:        int64(u.Id),
		Name:      u.Name,
		Email:     u.Email,
		Password:  u.Password,
		IsAdmin:   u.IsAdmin,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}
