package db

import (
	"context"

	"gorm.io/gorm"
	"linkit/internal/db/model"
)

type UserDao struct {
	store *DB
}

func (u *UserDao) FindByCredential(ctx context.Context, credential string) (*model.User, error) {
	var user model.User
	err := u.store.Client.WithContext(ctx).
		Where("username = ? OR email = ? OR nickname = ?", credential, credential, credential).
		First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := u.store.Client.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) GetByToken(ctx context.Context, token string) (*model.User, error) {
	var user model.User
	err := u.store.Client.WithContext(ctx).Where("token = ?", token).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (u *UserDao) UpdateToken(ctx context.Context, userID int64, token *string) error {
	return u.store.Client.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"token": token,
	}).Error
}

func (u *UserDao) UpdatePassword(ctx context.Context, userID int64, newHash string) error {
	return u.store.Client.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"password": newHash,
	}).Error
}
