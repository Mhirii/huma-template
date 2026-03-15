package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/dto"
	"github.com/mhirii/huma-template/internal/svc"
	"github.com/mhirii/huma-template/pkg/ctx"
	"github.com/rs/zerolog"
)

type UsersHandler struct {
	svc *svc.UsersService
	log zerolog.Logger
}

func RegisterUserRoutes(api huma.API, svc *svc.UsersService) {
	h := &UsersHandler{svc: svc}
	g := huma.NewGroup(api, "/users")
	g.UseSimpleModifier(func(op *huma.Operation) {
		op.Tags = []string{"Users"}
	})

	huma.Register(g, huma.Operation{
		OperationID:   "create-user",
		Method:        http.MethodPost,
		Path:          "/",
		Summary:       "Create a new User",
		Description:   "Create a new User",
		DefaultStatus: http.StatusOK,
	}, h.Create)

	huma.Register(g, huma.Operation{
		OperationID:   "get-user-by-id",
		Method:        http.MethodGet,
		Path:          "/{id}",
		Summary:       "Get a User",
		Description:   "Get a User by their ID, if you need to get a user by a different field you can use the other endpoint",
		DefaultStatus: http.StatusOK,
	}, h.Get)

	huma.Register(g, huma.Operation{
		OperationID:   "get-user-by-field",
		Method:        http.MethodGet,
		Path:          "/{id}",
		Summary:       "Get a user by field",
		Description:   "Get a User by any unique field",
		DefaultStatus: http.StatusOK,
	}, h.Get)

	huma.Register(g, huma.Operation{
		OperationID:   "update-user",
		Method:        http.MethodPut,
		Path:          "/{id}",
		Summary:       "Update a User",
		Description:   "Update a User",
		DefaultStatus: http.StatusOK,
	}, h.Update)

	huma.Register(g, huma.Operation{
		OperationID:   "delete-user",
		Method:        http.MethodDelete,
		Path:          "/{id}",
		Summary:       "Delete a User",
		Description:   "Delete a User",
		DefaultStatus: http.StatusOK,
	}, h.Delete)

	huma.Register(g, huma.Operation{
		OperationID:   "list-users",
		Method:        http.MethodGet,
		Path:          "/",
		Summary:       "List all Users",
		Description:   "List all Users",
		DefaultStatus: http.StatusOK,
	}, h.List)
}

func (h *UsersHandler) List(c context.Context, input *dto.ListUsersReq) (*dto.ListUsersRes, error) {
	ctx := ctx.FromContext(c)
	_ = ctx
	return nil, huma.Error501NotImplemented("Not implemented")
}

func (h *UsersHandler) Delete(c context.Context, input *dto.DeleteUserReq) (*dto.DeleteUserRes, error) {
	ctx := ctx.FromContext(c)
	_ = ctx
	return nil, huma.Error501NotImplemented("Not implemented")
}

func (h *UsersHandler) Update(c context.Context, input *dto.UpdateUserReq) (*dto.UpdateUserRes, error) {
	ctx := ctx.FromContext(c)
	_ = ctx
	return nil, huma.Error501NotImplemented("Not implemented")
}

func (h *UsersHandler) Get(c context.Context, input *dto.GetUserByIDReq) (*dto.GetUserByIDRes, error) {
	ctx := ctx.FromContext(c)
	_ = ctx
	return nil, huma.Error501NotImplemented("Not implemented")
}

func (h *UsersHandler) Create(c context.Context, input *dto.CreateUserReq) (*dto.CreateUserRes, error) {
	ctx := ctx.FromContext(c)
	_ = ctx
	return nil, huma.Error501NotImplemented("Not implemented")
}
