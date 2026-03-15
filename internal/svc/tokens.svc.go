package svc

import (
	"context"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/models"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/mhirii/huma-template/pkg/tokens"
	"github.com/oklog/ulid/v2"
	"github.com/uptrace/bun"
)

type TokensService struct {
	db *bun.DB
	tp *tokens.TokenProvider
}

func NewTokensService(tp *tokens.TokenProvider, db *bun.DB) *TokensService {
	return &TokensService{
		tp: tp,
		db: db,
	}
}

func (s *TokensService) TokensPair(ctx context.Context, sub string, username string, email string, tx *bun.Tx) (*tokens.TokensPair, error) {
	log := logging.FromCtx(ctx)
	tokenID := ulid.Make().String()
	t, err := s.tp.GetTokensPair(ctx, sub, username, email, tokenID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get tokens pair")
		return nil, huma.Error500InternalServerError("could not create tokens")
	}
	m := models.RefreshTokens{
		RTokenID:  tokenID,
		UserID:    sub,
		Token:     t.RefreshTokenID,
		Device:    "",
		ExpiresAt: t.RefreshExp,
		RevokedAt: nil,
	}
	newInsert := s.db.NewInsert()
	if tx != nil {
		newInsert = tx.NewInsert()
	}
	if err := newInsert.Model(&m).Returning("*").Scan(ctx, &m); err != nil {
		log.Error().Err(err).Msg("failed to insert refresh token")
		return nil, huma.Error500InternalServerError("could not save token")
	}

	return t, nil
}

func (s *TokensService) RefreshTokens(ctx context.Context, refreshToken string) (string, uint, error) {
	log := logging.FromCtx(ctx)
	claims, err := s.tp.ParseRefresh(ctx, refreshToken)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse refresh token")
		return "", 0, huma.Error500InternalServerError("could not parse refresh token")
	}
	t := models.RefreshTokens{RTokenID: claims.TokenID}
	err = s.db.NewSelect().Model(&t).WherePK("id").Scan(ctx, &t)
	if err != nil {
		log.Error().Err(err).Msg("failed to select refresh token")
		return "", 0, huma.Error500InternalServerError("could not select refresh token")
	}
	if t.RevokedAt != nil {
		// TODO: log the user out of all sessions
		return "", 0, huma.Error400BadRequest("refresh token has been revoked")
	}

	aToken, aexp, err := s.tp.GetAccess(ctx, claims.Subject, claims.Username, claims.Email, claims.TokenID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get access token")
		return "", 0, huma.Error500InternalServerError("could not create access token")
	}
	return aToken.String(), uint(aexp.Unix()), nil
}

func (s *TokensService) ValidateAccessToken(ctx context.Context, token string) (*tokens.UserClaims, error) {
	log := logging.FromCtx(ctx)
	claims, err := s.tp.ParseAccess(ctx, token)
	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			return nil, huma.Error400BadRequest(err.Error())
		} else if strings.Contains(err.Error(), "invalid token type") {
			log.Err(err).Msg(err.Error())
			return nil, huma.Error400BadRequest("invalid token type")
		}
		log.Error().Err(err).Msg("failed to parse access token")
		return nil, huma.Error500InternalServerError("could not parse access token")
	}

	t := models.RefreshTokens{RTokenID: claims.TokenID}
	if err := s.db.NewSelect().Model(&t).WherePK("id").Scan(ctx, &t); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, huma.Error401Unauthorized("invalid token")
		}
		return nil, huma.Error500InternalServerError("could not select refresh token")
	}
	if t.RevokedAt != nil {
		// TODO: log the user out of all sessions
		return nil, huma.Error401Unauthorized("refresh token has been revoked")
	}
	return claims, nil
}

func (s *TokensService) RevokeRefreshToken(ctx context.Context, token string) error {
	log := logging.FromCtx(ctx)
	claims, err := s.tp.ParseRefresh(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse refresh token")
		return huma.Error500InternalServerError("could not parse refresh token")
	}
	revokedAt := time.Now()
	t := models.RefreshTokens{RTokenID: claims.TokenID, RevokedAt: &revokedAt}
	if _, err := s.db.NewUpdate().Model(&t).OmitZero().WherePK("id").Exec(ctx); err != nil {
		log.Error().Err(err).Msg("failed to revoke refresh token")
		return huma.Error500InternalServerError("could not revoke refresh token")
	}

	return nil
}
