package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/dto"
	"github.com/mhirii/huma-template/internal/svc"
	"github.com/mhirii/huma-template/pkg/ctx"
	"github.com/rs/zerolog"
)

type AuthHandler struct {
	svc *svc.AuthService
	log zerolog.Logger
}

func RegisterAuthRoutes(api huma.API, svc *svc.AuthService) {
	h := &AuthHandler{svc: svc}
	g := huma.NewGroup(api, "/auth")
	g.UseSimpleModifier(func(op *huma.Operation) {
		op.Tags = []string{"Auth"}
	})

	huma.Register(g, huma.Operation{
		OperationID:   "login",
		Method:        http.MethodPost,
		Path:          "/login",
		Summary:       "Login",
		Description:   "Login",
		DefaultStatus: http.StatusOK,
	}, h.Login)

	huma.Register(g, huma.Operation{
		OperationID:   "signup",
		Method:        http.MethodPost,
		Path:          "/signup",
		Summary:       "Signup",
		Description:   "Create a new Account",
		DefaultStatus: http.StatusOK,
	}, h.Signup)

	huma.Register(g, huma.Operation{
		OperationID:   "refresh",
		Method:        http.MethodPost,
		Path:          "/refresh",
		Summary:       "Refresh",
		Description:   "Refresh your Access Token",
		DefaultStatus: http.StatusOK,
	}, h.Refresh)

	huma.Register(g, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          "/logout",
		Summary:       "Logout",
		Description:   "Logout and revoke your Tokens",
		DefaultStatus: http.StatusOK,
	}, h.Logout)

	huma.Register(g, huma.Operation{
		OperationID:   "verify",
		Method:        http.MethodPost,
		Path:          "/verify",
		Summary:       "Verify",
		Description:   "Verify a Token",
		DefaultStatus: http.StatusOK,
	}, h.Verify)
}

func (h *AuthHandler) Login(c context.Context, input *dto.LoginReq) (*dto.LoginRes, error) {
	ctx := ctx.FromContext(c)
	u, t, err := h.svc.Login(ctx, *input.Body.Username, input.Body.Password)
	if err != nil {
		return nil, err
	}
	return &dto.LoginRes{
		Body: dto.LoginResBody{
			Tokens: dto.Tokens{
				AccessAndExp: dto.AccessAndExp{
					AccessToken:          t.AccessToken.String(),
					AccessTokenExpiresAt: uint(t.AccessExp.Unix()),
				},
				RefreshAndExp: dto.RefreshAndExp{
					RefreshToken:          t.RefreshToken.String(),
					RefreshTokenExpiresAt: uint(t.RefreshExp.Unix()),
				},
			},
			User: *u,
		},
	}, nil
}

func (h *AuthHandler) Signup(c context.Context, input *dto.SignupReq) (*dto.SignupRes, error) {
	ctx := ctx.FromContext(c)
	_, t, err := h.svc.Signup(ctx, *input)
	if err != nil {
		return nil, err
	}
	return &dto.SignupRes{
		Body: dto.Tokens{
			AccessAndExp: dto.AccessAndExp{
				AccessToken:          t.AccessToken.String(),
				AccessTokenExpiresAt: uint(t.AccessExp.Unix()),
			},
			RefreshAndExp: dto.RefreshAndExp{
				RefreshToken:          t.RefreshToken.String(),
				RefreshTokenExpiresAt: uint(t.RefreshExp.Unix()),
			},
		},
	}, nil
}

func (h *AuthHandler) Refresh(c context.Context, input *dto.RefreshReq) (*dto.RefreshRes, error) {
	ctx := ctx.FromContext(c)
	token, exp, err := h.svc.Refresh(ctx, input.Body.RefreshToken)
	if err != nil {
		return nil, err
	}
	return &dto.RefreshRes{
		Body: dto.AccessAndExp{
			AccessToken:          token,
			AccessTokenExpiresAt: exp,
		},
	}, nil
}
func (h *AuthHandler) Logout(c context.Context, input *dto.LogoutReq) (*dto.LogoutRes, error) {
	ctx := ctx.FromContext(c)
	if input.Authorization == "" {
		return &dto.LogoutRes{Body: struct{}{}}, huma.NewError(http.StatusBadRequest, "Authorization header is required")
	}
	sp := strings.Split(input.Authorization, " ")
	if len(sp) != 2 {
		return &dto.LogoutRes{Body: struct{}{}}, huma.NewError(http.StatusBadRequest, "Authorization header is invalid")
	}
	if sp[0] != "Bearer" {
		return &dto.LogoutRes{Body: struct{}{}}, huma.NewError(http.StatusBadRequest, "Authorization header is invalid")
	}
	token := sp[1]
	err := h.svc.Logout(ctx, token)
	return &dto.LogoutRes{Body: struct{}{}}, err
}

func (h *AuthHandler) Verify(c context.Context, input *dto.VerifyReq) (*dto.VerifyRes, error) {
	ctx := ctx.FromContext(c)
	claims, err := h.svc.Verify(ctx, input.Body.Token)
	if err != nil {
		return nil, err
	}
	return &dto.VerifyRes{Body: claims}, err
}
