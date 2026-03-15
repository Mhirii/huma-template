package ctx

import (
	"context"

	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/rs/zerolog"
)

type ServiceContext struct {
	context.Context
	Log zerolog.Logger
}

func NewServiceContext(ctx context.Context, log zerolog.Logger) ServiceContext {
	return ServiceContext{
		Context: ctx,
		Log:     log,
	}
}

func (c ServiceContext) GetLogger() zerolog.Logger {
	return c.Log
}

func (c ServiceContext) SetLogger(log zerolog.Logger) ServiceContext {
	c.Log = log
	return c
}

func FromContext(ctx context.Context) ServiceContext {
	c := ctx
	log := c.Value("logger").(zerolog.Logger)
	if &log == nil {
		l := logging.L()
		log = l
		c = context.WithValue(ctx, "logger", log)
	}
	return ServiceContext{
		Context: c,
		Log:     log,
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
