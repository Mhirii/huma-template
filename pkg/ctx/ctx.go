package ctx

import (
	"context"

	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/rs/zerolog"
)

type ServiceContext struct {
	context.Context
	Log    zerolog.Logger
	UserID *string
}

func NewServiceContext(ctx context.Context, log zerolog.Logger) ServiceContext {
	return ServiceContext{
		Context: ctx,
		Log:     log,
		UserID:  nil,
	}
}

func (c ServiceContext) GetLogger() zerolog.Logger {
	return c.Log
}

func (c ServiceContext) SetLogger(log zerolog.Logger) ServiceContext {
	c.Log = log
	return c
}

func (c ServiceContext) GetUserID() *string {
	return c.UserID
}

func FromContext(ctx context.Context) ServiceContext {
	c := ctx
	log := c.Value("logger").(zerolog.Logger)
	userID := ""
	if &log == nil {
		l := logging.L()
		log = l
		c = context.WithValue(ctx, "logger", log)
	}
	_userID := c.Value("userID")
	if _userID == nil {
		userID = ""
	} else {
		userID = _userID.(string)
		c = context.WithValue(ctx, "userID", userID)
	}
	return ServiceContext{
		Context: c,
		Log:     log,
		UserID:  &userID,
	}
}

func fromSvcContext(ctx ServiceContext) ServiceContext {
	return ServiceContext{
		Context: ctx.Context,
		Log:     ctx.Log,
	}
}

func WithValue(ctx context.Context, key, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}
