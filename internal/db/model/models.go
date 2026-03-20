package model

import "time"

type User struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"column:username;type:text;not null;uniqueIndex" json:"username"`
	Password  string    `gorm:"column:password;type:text;not null" json:"-"`
	Email     string    `gorm:"column:email;type:text;not null;uniqueIndex" json:"email"`
	Nickname  string    `gorm:"column:nickname;type:text;not null" json:"nickname"`
	Token     *string   `gorm:"column:token;type:text" json:"token"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

type Resource struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Filename  string    `gorm:"column:filename;type:text;not null" json:"filename"`
	Hash      string    `gorm:"column:hash;type:text;not null" json:"hash"`
	Type      string    `gorm:"column:type;type:text;not null" json:"type"`
	Path      string    `gorm:"column:path;type:text;not null" json:"path"`
	FileSize  int64     `gorm:"column:file_size;not null;default:0" json:"fileSize"`
	UserID    int64     `gorm:"column:user_id;not null;index" json:"user_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type AppConfig struct {
	ID    int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Key   string  `gorm:"column:key;type:text;not null;uniqueIndex" json:"key"`
	Value *string `gorm:"column:value;type:text" json:"value"`
}

type ResourceTag struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ResourceID int64     `gorm:"column:resource_id;not null;uniqueIndex:idx_resource_tag_resource_id_tag,priority:1;index:idx_resource_tag_tag" json:"resource_id"`
	Tag        string    `gorm:"column:tag;type:text;not null;uniqueIndex:idx_resource_tag_resource_id_tag,priority:2;index:idx_resource_tag_tag" json:"tag"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

type Share struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ResourceID int64      `gorm:"column:resource_id;not null;index" json:"resource_id"`
	UserID     int64      `gorm:"column:user_id;not null;index" json:"user_id"`
	Code       string     `gorm:"column:code;type:text;not null;uniqueIndex" json:"code"`
	Password   *string    `gorm:"column:password;type:text" json:"-"`
	ExpireTime *time.Time `gorm:"column:expire_time" json:"-"`
	Relay      bool       `gorm:"column:relay;not null;default:false" json:"relay"`
	ViewCount  int64      `gorm:"column:view_count;not null;default:0" json:"viewCount"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	Resource   Resource   `gorm:"foreignKey:ResourceID;references:ID" json:"-"`
}

type ShareCode struct {
	ID         int64     `json:"id"`
	ResourceID int64     `json:"resource_id"`
	UserID     int64     `json:"user_id"`
	Code       string    `json:"code"`
	ViewCount  int64     `json:"viewCount"`
	Relay      bool      `json:"relay"`
	CreatedAt  time.Time `json:"created_at"`
}

// 用于列表返回携带分享码

type UserResourceWithShare struct {
	ID        int64     `json:"id"`
	Filename  string    `json:"filename"`
	Type      string    `json:"type"`
	Storage   string    `json:"storage"`
	CreatedAt time.Time `json:"createdAt"`
	ShareCode *string   `json:"shareCode"`
	Tags      []string  `json:"tags"`
}

type ShareResource struct {
	ShareID    int64      `json:"shareId"`
	Code       string     `json:"code"`
	ResourceID int64      `json:"resourceId"`
	UserID     int64      `json:"-"`
	Filename   string     `json:"filename"`
	Path       string     `json:"path"`
	Type       string     `json:"type"`
	Relay      bool       `json:"relay"`
	ViewCount  int64      `json:"viewCount"`
	CreatedAt  time.Time  `json:"createdAt"`
	Password   *string    `json:"-"`
	ExpireTime *time.Time `json:"-"`
}

func (User) TableName() string {
	return "user"
}

func (Resource) TableName() string {
	return "resource"
}

func (AppConfig) TableName() string {
	return "app_config"
}

func (ResourceTag) TableName() string {
	return "resource_tag"
}

func (Share) TableName() string {
	return "share"
}
