package svc

import (
	"fmt"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mhirii/huma-template/internal/dto"
	"github.com/mhirii/huma-template/internal/models"
	"github.com/mhirii/huma-template/pkg/ctx"
	"github.com/mhirii/huma-template/pkg/logging"
	"github.com/oklog/ulid/v2"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type UsersService struct {
	db *bun.DB
}

func NewUsersService(db *bun.DB) (*UsersService, error) {
	return &UsersService{db: db}, nil
}

func (s *UsersService) GetUsers(ctx ctx.ServiceContext, params *dto.ListUsersReq) (*dto.ListUsersRes, error) {
	log := logging.FromCtx(ctx)
	log.Debug().
		Int("page", params.Page).
		Int("per_page", params.PerPage).
		Str("sort_by", params.SortBy).
		Str("sort_dir", params.SortDir).
		Str("search", params.Search).
		Str("filters", params.Filters).
		Str("includes", params.Includes).
		Msg("listing users")

	var users []models.Users
	res := &dto.ListUsersRes{
		Body: dto.ListUsersResBody{
			Total:     0,
			ListQuery: dto.ListQueryRes{},
			Users:     nil,
		},
	}

	q := s.db.NewSelect().Model(&users)

	if params.Search != "" {
		search := "%" + params.Search + "%"
		q = q.Where(
			"(username ILIKE ? OR email ILIKE ? OR first_name ILIKE ? OR family_name ILIKE ?)",
			search, search, search, search,
		)
	}

	filters, err := dto.ParseFilters(params.Filters)
	if err != nil {
		log.Warn().Err(err).Str("filters", params.Filters).Msg("invalid filters, ignoring")
	}
	q = dto.ApplyFilters(filters, q)

	total, err := q.Clone().Count(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to count users")
		return nil, huma.Error500InternalServerError(err.Error())
	}
	res.Body.Total = total
	log.Info().Int("total", total).Msg("counted users")

	q = q.Order(params.SortBy + " " + params.SortDir)
	q = q.Limit(params.PerPage)
	q = q.Offset(params.PerPage * (params.Page - 1))

	if err := q.Scan(ctx, &users); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Debug().Msg("no users found for query")
			return res, nil
		}
		log.Error().Err(err).Msg("failed to scan users")
		return nil, huma.Error500InternalServerError(err.Error())
	}

	resUsers := []dto.UserModelRes{}
	for _, u := range users {
		resUsers = append(resUsers, *s.ModelToRes(&u))
	}
	res.Body.Users = resUsers
	res.Body.ListQuery = dto.ListQueryRes{
		Page: params.Page, PerPage: params.PerPage, SortBy: params.SortBy, SortDir: params.SortDir, Search: params.Search, Includes: params.Includes, Filters: filters,
	}
	log.Info().Int("results", len(resUsers)).Msg("retrieved users")
	return res, nil
}

func (s *UsersService) GetUserByID(ctx ctx.ServiceContext, id string) (*dto.UserModelRes, error) {
	log := logging.FromCtx(ctx)
	log.Debug().Str("user_id", id).Msg("fetching user by id")
	m := models.Users{UserID: id}
	if err := s.db.NewSelect().Model(&m).WherePK("id").Scan(ctx, &m); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Warn().Str("user_id", id).Msg("user not found")
		} else {
			log.Error().Err(err).Str("user_id", id).Msg("failed to fetch user")
		}
		return nil, huma.Error404NotFound("user not found")
	}
	return s.ModelToRes(&m), nil
}

func (s *UsersService) GetUserByField(ctx ctx.ServiceContext, f string, v string) (*dto.UserModelRes, error) {
	log := logging.FromCtx(ctx)
	m := models.Users{}
	q := s.db.NewSelect().Model(&m).Where(fmt.Sprintf("%s = ?", f), v)
	log.Debug().Str("field", f).Str("value", v).Msg("looking up user by field")
	if err := q.Scan(ctx, &m); err != nil {
		log.Error().Err(err).Str("field", f).Str("value", v).Str("query", q.String()).Msg("failed to get user by field")
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return s.ModelToRes(&m), nil
}

func (s *UsersService) CreateUser(c ctx.ServiceContext, data *dto.CreateUserReqBody, tx *bun.Tx) (*dto.UserModelRes, error) {
	log := logging.FromCtx(c)
	log = log.With().Str("username", data.Username).Str("email", data.Email).Logger()
	c = ctx.NewServiceContext(c, log)
	log.Debug().Msg("creating user")

	userID := ulid.Make().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("failed to hash password")
		return nil, huma.Error500InternalServerError(err.Error())
	}
	m := models.Users{
		Username:     data.Username,
		Email:        data.Email,
		UserID:       userID,
		PasswordHash: string(hash),
	}
	newInsert := s.db.NewInsert()
	if tx != nil {
		newInsert = tx.NewInsert()
	}
	if _, err := newInsert.Model(&m).Returning("*").Exec(c, &m); err != nil {
		log.Error().Err(err).Msg("failed to insert user")
		if strings.Contains(err.Error(), "users_username_key") {
			log.Warn().Err(err).Msg("username already exists")
			return nil, huma.Error400BadRequest("username already exists")
		}
		if strings.Contains(err.Error(), "users_email_key") {
			log.Warn().Err(err).Msg("email already exists")
			return nil, huma.Error400BadRequest("email already exists")
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	log.Info().Str("user_id", m.UserID).Msg("created user")
	return s.ModelToRes(&m), nil
}

func (s *UsersService) UpdateUser(ctx ctx.ServiceContext, user models.Users) (*dto.UserModelRes, error) {
	log := logging.FromCtx(ctx)
	log.Debug().Str("user_id", user.UserID).Msg("updating user")
	m := user
	m.UserID = user.UserID
	if err := s.db.NewUpdate().Model(&m).Returning("*").OmitZero().WherePK("id").Scan(ctx, &m); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Warn().Str("user_id", user.UserID).Msg("user not found for update")
			return nil, huma.Error404NotFound("user not found")
		}
		log.Error().Err(err).Str("user_id", user.UserID).Msg("failed to update user")
		return nil, huma.Error500InternalServerError(err.Error())
	}
	log.Info().Str("user_id", user.UserID).Msg("updated user")
	return s.ModelToRes(&m), nil
}

func (s *UsersService) DeleteUser(ctx ctx.ServiceContext, id string) error {
	log := logging.FromCtx(ctx)
	log.Debug().Str("user_id", id).Msg("deleting user")
	m := models.Users{UserID: id}
	if _, err := s.db.NewDelete().Model(&m).WherePK("id").Exec(ctx); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			log.Warn().Str("user_id", id).Msg("user not found for delete")
			return huma.Error404NotFound("user not found")
		}
		log.Error().Err(err).Str("user_id", id).Msg("failed to delete user")
		return huma.Error500InternalServerError(err.Error())
	}
	log.Info().Str("user_id", id).Msg("deleted user")
	return nil
}

func (s *UsersService) ModelToRes(m *models.Users) *dto.UserModelRes {
	return UsersModelToRes(m)
}

func UsersModelToRes(m *models.Users) *dto.UserModelRes {
	if m == nil {
		return nil
	}
	res := &dto.UserModelRes{}
	res.ID = m.UserID
	res.Username = m.Username
	res.Email = m.Email

	if m.AvatarURL != "" {
		res.Avatar = &m.AvatarURL
	}

	if !m.CreatedAt.IsZero() {
		res.CreatedAt = int(m.CreatedAt.Unix())
	}
	if !m.UpdatedAt.IsZero() {
		res.UpdatedAt = int(m.UpdatedAt.Unix())
	}
	return res
}
