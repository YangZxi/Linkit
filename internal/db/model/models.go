package model

import "time"

type User struct {
	ID        int64     `db:"id" json:"id"`
	Username  string    `db:"username" json:"username"`
	Password  string    `db:"password" json:"-"`
	Email     string    `db:"email" json:"email"`
	Nickname  string    `db:"nickname" json:"nickname"`
	Token     *string   `db:"token" json:"token"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Resource struct {
	ID        int64     `db:"id" json:"id"`
	Filename  string    `db:"filename" json:"filename"`
	Hash      string    `db:"hash" json:"hash"`
	Type      string    `db:"type" json:"type"`
	Path      string    `db:"path" json:"path"`
	FileSize  int64     `db:"file_size" json:"fileSize"`
	UserID    int64     `db:"user_id" json:"user_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ShareCode struct {
	ID         int64     `db:"id" json:"id"`
	ResourceID int64     `db:"resource_id" json:"resource_id"`
	UserID     int64     `db:"user_id" json:"user_id"`
	Code       string    `db:"code" json:"code"`
	ViewCount  int64     `db:"view_count" json:"viewCount"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// 用于列表返回携带分享码

type UserResourceWithShare struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	ShareCode *string   `json:"shareCode"`
}

type ShareResource struct {
	ShareID    int64     `json:"share_id"`
	Code       string    `json:"code"`
	ResourceID int64     `json:"resource_id"`
	Filename   string    `json:"filename"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	ViewCount  int64     `json:"viewCount"`
	CreatedAt  time.Time `json:"created_at"`
}
