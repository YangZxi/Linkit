package db

import (
	"context"
	"database/sql"
	"errors"

	"linkit/internal/db/model"
)

type UserDao struct {
	store *DB
}

func (u *UserDao) FindByCredential(ctx context.Context, credential string) (*model.User, error) {
	row := u.store.Client.QueryRowContext(ctx, `SELECT id, username, password, email, nickname, token, created_at, updated_at FROM user WHERE username = ? OR email = ? OR nickname = ? LIMIT 1`, credential, credential, credential)
	var user model.User
	if err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Nickname, &user.Token, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) GetByToken(ctx context.Context, token string) (*model.User, error) {
	row := u.store.Client.QueryRowContext(ctx, `SELECT id, username, password, email, nickname, token, created_at, updated_at FROM user WHERE token = ? LIMIT 1`, token)
	var user model.User
	if err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Nickname, &user.Token, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) UpdateToken(ctx context.Context, userID int64, token *string) error {
	_, err := u.store.Client.ExecContext(ctx, `UPDATE user SET token = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, token, userID)
	return err
}

func (u *UserDao) UpdatePassword(ctx context.Context, userID int64, newHash string) error {
	_, err := u.store.Client.ExecContext(ctx, `UPDATE user SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, newHash, userID)
	return err
}
