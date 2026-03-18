package svc

import (
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/dto"
	"github.com/mhirii/huma-template/internal/models"
	"github.com/mhirii/huma-template/pkg/ctx"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/mhirii/huma-template/pkg/tokens"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db   *bun.DB
	log  zerolog.Logger
	usvc *UsersService
	tsvc *TokensService
}

func NewAuthService(usvc *UsersService, tsvc *TokensService, db *bun.DB) (*AuthService, error) {
	return &AuthService{usvc: usvc, tsvc: tsvc, db: db}, nil
}

type Tokens struct {
	Access     string    `json:"access"`
	AccessExp  time.Time `json:"access_exp"`
	Refresh    string    `json:"refresh"`
	RefreshExp time.Time `json:"refresh_exp"`
}

func (s *AuthService) Signup(ctx ctx.ServiceContext, data dto.SignupReq) (*dto.UserModelRes, *tokens.TokensPair, error) {
	log := logging.FromCtx(ctx)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to start transaction")
		return nil, nil, huma.Error500InternalServerError(err.Error())
	}
	defer tx.Rollback()
	u, err := s.usvc.CreateUser(ctx, &dto.CreateUserReqBody{
		Email:    data.Body.Email,
		Username: data.Body.Username,
		Password: data.Body.Password,
	}, &tx)
	if err != nil {
		return nil, nil, err
	}

	t, err := s.tsvc.TokensPair(ctx, u.ID, u.Username, u.Email, &tx)
	if err != nil {
		s.log.Err(err).Msg("could not create tokens")
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
		return nil, nil, huma.Error500InternalServerError(err.Error())
	}
	return u, t, nil
}

func (s *AuthService) Login(ctx ctx.ServiceContext, identifier, password string) (*dto.UserModelRes, *tokens.TokensPair, error) {
	log := logging.FromCtx(ctx)

	u := models.Users{}

	err := s.db.NewSelect().Model(&u).WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
		return q.WhereOr("username = ?", identifier).
			WhereOr("email = ?", identifier)
	}).Scan(ctx, &u)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		return nil, nil, huma.Error500InternalServerError("failed to get user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil, huma.Error401Unauthorized("invalid credentials")
	}

	t, err := s.tsvc.TokensPair(ctx, u.UserID, u.Username, u.Email, nil)
	if err != nil {
		log.Err(err).Msg("could not create tokens")
		return nil, nil, huma.Error500InternalServerError(err.Error())
	}
	return UsersModelToRes(&u), t, nil
}

func (s *AuthService) Refresh(ctx ctx.ServiceContext, token string) (string, uint, error) {
	return s.tsvc.RefreshTokens(ctx, token)
}

func (s *AuthService) Logout(ctx ctx.ServiceContext, token string) error {
	return s.tsvc.RevokeRefreshToken(ctx, token)
}

func (s *AuthService) Revoke(ctx ctx.ServiceContext, token string) error {
	return s.tsvc.RevokeRefreshToken(ctx, token)
}

func (s *AuthService) Verify(ctx ctx.ServiceContext, token string) (*tokens.UserClaims, error) {
	return s.tsvc.ValidateAccessToken(ctx, token)
}
