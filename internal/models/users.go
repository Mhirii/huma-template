package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Users struct {
	bun.BaseModel `bun:"table:users,alias:u"`
	Email         string `bun:"email"`
	EmailVerified bool   `bun:"email_verified"`
	PasswordHash  string `bun:"password_hash"`
	Username      string `bun:"username"`
	AvatarURL     string `bun:"avatar_url"`

	CreatedAt time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt time.Time `bun:"updated_at,default:current_timestamp"`

	UserID string `bun:"id,pk"`
}
