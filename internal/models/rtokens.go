package models

import (
	"time"

	"github.com/uptrace/bun"
)

type RefreshTokens struct {
	bun.BaseModel `bun:"table:refresh_tokens,alias:rt"`
	Token         string     `bun:"token"`
	Device        string     `bun:"device"`
	ExpiresAt     time.Time  `bun:"expires_at,default:current_timestamp"`
	CreatedAt     time.Time  `bun:"created_at,default:current_timestamp"`
	RevokedAt     *time.Time `bun:"revoked_at,default:current_timestamp"`

	RTokenID string `bun:"id,pk"`
	UserID   string `bun:"user_id"`
	User     *Users `bun:"rel:belongs-to,join:user_id=id"`
}
