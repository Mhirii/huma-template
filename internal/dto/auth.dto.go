package dto

import "github.com/mhirii/huma-template/pkg/tokens"

type AccessAndExp struct {
	AccessToken          string `json:"access_token" doc:"The access token"`
	AccessTokenExpiresAt uint   `json:"access_token_expires_at" doc:"The access token expiration time in unix time (seconds)"`
}
type RefreshAndExp struct {
	RefreshToken          string `json:"refresh_token" doc:"The refresh token"`
	RefreshTokenExpiresAt uint   `json:"refresh_token_expires_at" doc:"The refresh token expiration time in unix time (seconds)"`
}
type Tokens struct {
	AccessAndExp
	RefreshAndExp
}

type LoginReq struct {
	Body struct {
		Username *string `json:"username,omitempty" doc:"Username of the user, either this or email is required" minLength:"3" MaxLength:"255"`
		Email    *string `json:"email,omitempty" doc:"Email of the user, either this or username is required" Email:"true" format:"email"`
		Password string  `json:"password" doc:"Password of the user" minLength:"8" MaxLength:"255" required:"true"`
	}
}
type LoginRes struct{ Body LoginResBody }
type LoginResBody struct {
	Tokens Tokens       `json:"tokens" doc:"Tokens"`
	User   UserModelRes `json:"user" doc:"User Model"`
}

type SignupReq struct {
	Body struct {
		Username string `json:"username" doc:"username of the user" minLength:"3" MaxLength:"255" required:"true"`
		Email    string `json:"email" doc:"Email of the user" Email:"true" required:"true" format:"email"`
		Password string `json:"password" doc:"Password of the user" minLength:"8" MaxLength:"255" required:"true"`
	}
}
type SignupRes struct{ Body Tokens }

type RefreshReq struct {
	Body struct {
		RefreshToken string `json:"refresh_token" doc:"Refresh token of the user" required:"true"`
	}
}
type RefreshRes struct{ Body AccessAndExp }

type LogoutReq struct {
	Authorization string `header:"Authorization" doc:"Bearer Token of the user" required:"true"`
}
type LogoutRes struct{ Body struct{} }

type VerifyReq struct {
	Body struct {
		Token string `json:"token" doc:"Token to verify" required:"true"`
	}
}
type VerifyRes struct{ Body *tokens.UserClaims }
