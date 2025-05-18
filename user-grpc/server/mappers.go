package server

import (
	"github.com/rx3lixir/user-service/pkg/password"
	pb "github.com/rx3lixir/user-service/user-grpc/gen/go"

	"time"

	"github.com/rx3lixir/user-service/internal/db"
)

func toStorerUser(u *pb.UserReq) *db.User {
	return &db.User{
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
		IsAdmin:  u.IsAdmin,
	}
}

func toPBUserRes(u *db.User) *pb.UserRes {
	return &pb.UserRes{
		Id:       int64(u.Id),
		Name:     u.Name,
		Email:    u.Email,
		Password: u.Password,
		IsAdmin:  u.IsAdmin,
	}
}

func patchUserReq(user *db.User, u *pb.UserReq) {
	if u.Name != "" {
		user.Name = u.Name
	}
	if u.Email != "" {
		user.Email = u.Email
	}
	if u.Password != "" {
		hashed, err := password.Hash(u.Password)
		if err != nil {
			panic(err)
		}
		user.Password = hashed
	}
	if u.IsAdmin {
		user.IsAdmin = u.IsAdmin
	}
	user.UpdatedAt = (time.Now())
}
