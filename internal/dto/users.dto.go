package dto

type CreateUserReq struct {
	AuthHeader
	Body CreateUserReqBody
}
type CreateUserReqBody struct {
	Username string `json:"username" doc:"Username of the user" minLength:"3" maxLength:"255" required:"true"`
	Email    string `json:"email" doc:"Email of the user" Email:"true" required:"true" format:"email"`
	Password string `json:"password" doc:"Password of the user" minLength:"8" maxLength:"255" required:"true"`
}

type CreateUserRes struct{ Body CreateUserResBody }
type CreateUserResBody struct {
	UserModelRes
}

type UpdateUserReq struct {
	AuthHeader
	Body struct {
		Id       string  `json:"id" doc:"ID of the user" required:"true"`
		Username *string `json:"username" doc:"username of the user" minLength:"3" maxLength:"255" required:"false"`
		Email    *string `json:"email" doc:"Email of the user" format:"email" required:"false"`
		Avatar   *string `json:"avatar" doc:"Avatar of the user" minLength:"3" maxLength:"255" required:"false"`
	}
}
type UpdateUserRes struct{ Body UpdateUserResBody }
type UpdateUserResBody struct {
	UserModelRes
}

type GetUserByFieldReq struct {
	AuthHeader
	Field string `path:"field" doc:"Field to find the user by" enum:"username,email,id" required:"true"`
	Value string `path:"value" doc:"Value to find the user by" required:"true"`
}
type GetUserByFieldRes struct{ Body UserModelRes }

type GetUserByIDReq struct {
	AuthHeader
	ID string `path:"id" doc:"ID of the user" required:"true"`
}
type GetUserByIDRes struct{ Body UserModelRes }

type DeleteUserReq struct {
	AuthHeader
	ID string `path:"id" doc:"ID of the user" required:"true"`
}

type DeleteUserResBody struct {
	ID string `json:"id"`
}
type DeleteUserRes struct {
	Body DeleteUserResBody
}

type ListUsersReq struct {
	AuthHeader
	ListQuery
}

type ListUsersResBody struct {
	Users     []UserModelRes `json:"users"`
	Total     int            `json:"total"`
	ListQuery ListQueryRes   `json:"query"`
}

type ListUsersRes struct {
	Body ListUsersResBody
}

type MeReq struct{ AuthHeader }
type MeRes struct {
	Body UserModelRes
}

type UserModelRes struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	FirstName string                 `json:"first_name,omitempty"`
	LastName  string                 `json:"last_name,omitempty"`
	Avatar    *string                `json:"avatar,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`

	CreatedAt int `json:"created_at"`
	UpdatedAt int `json:"updated_at"`
}
