package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *PostgresStore) CreateUser(parentCtx context.Context, user *User) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	query := `
		INSERT INTO users (name, email, password, is_admin)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.Password,
		user.IsAdmin,
	).Scan(&user.Id, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("Failed to create user: %w", err)
	}

	return nil
}

func (s *PostgresStore) UpdateUser(parentCtx context.Context, user *User) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	var exists bool

	// Проверка есть ли запрашиваемый пользователь
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", user.Id).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("user with ID %d not found", user.Id)
	}

	query := `
		UPDATE users
		SET name = $1, email = $2, password = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING updated_at
	`
	err = s.db.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.Password,
		user.Id).Scan(&user.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update user %d: %w", user.Id, err)
	}

	return err
}

func (s *PostgresStore) GetUsers(parentCtx context.Context) ([]*User, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	rows, err := s.db.Query(ctx, "SELECT * FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		user, err := scanIntoUser(rows)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}

func (s *PostgresStore) GetUserByID(parentCtx context.Context, id int) (*User, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	row := s.db.QueryRow(ctx, "SELECT id, name, email, password, is_admin, created_at, updated_at FROM users WHERE id = $1", id)

	user := new(User)
	err := row.Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user %d not found", id)
		}
		return nil, fmt.Errorf("failed to get user by id %d: %w", id, err)
	}

	return user, nil
}

func (s *PostgresStore) GetUserByEmail(parentCtx context.Context, email string) (*User, error) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	row := s.db.QueryRow(ctx, "SELECT id, name, email, password, is_admin, created_at, updated_at FROM users WHERE email = $1", email)

	user := new(User)
	err := row.Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user %v not found", email)
		}
		return nil, fmt.Errorf("failed to get user by email %v: %w", email, err)
	}

	return user, nil
}

func (s *PostgresStore) DeleteUser(parentCtx context.Context, id int) error {
	ctx, cancel := context.WithTimeout(parentCtx, time.Second*3)
	defer cancel()

	cmdTag, err := s.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failedt to delete user %d: %w", id, err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user with ID %d not found for deletion", id)
	}

	return nil
}

func scanIntoUser(rows pgx.Rows) (*User, error) {
	user := new(User)

	err := rows.Scan(
		&user.Id,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, err
}
