package dto

type GetAllUsersQuery struct {
	Page  int    `query:"page"`
	Limit int    `query:"limit"`
	Sort  string `query:"sort"`
	Role  string `query:"role"`
	Name  string `query:"name"`
	Email string `query:"email"`
}

type GetUserByIDUri struct {
	ID uint `param:"id"`
}

type CreateUserBody struct {
	Name     string `json:"name"     validate:"required,min=1,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Role     string `json:"role"     validate:"required,oneof=admin guest manager"`
	Password string `json:"password" validate:"required,min=6,max=255"`
}

type UpdateUserUri struct {
	ID uint `param:"id"`
}

type UpdateUserBody struct {
	Name     string `json:"name"     validate:"required,min=1,max=100"`
	Email    string `json:"email"    validate:"required,email"`
	Role     string `json:"role"     validate:"required,oneof=admin guest manager"`
	Password string `json:"password" validate:"omitempty,min=6,max=255"`
}

type DeleteUserUri struct {
	ID uint `param:"id"`
}
