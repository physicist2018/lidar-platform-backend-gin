package dto

type GetAllUsersQuery struct {
	Page  int    `form:"page"  binding:"omitempty,min=1"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort  string `form:"sort"  binding:"omitempty,oneof=asc desc"`
	Role  string `form:"role"  binding:"omitempty,oneof=admin guest manager"`
	Name  string `form:"name"`
	Email string `form:"email"`
}

type GetUserByIDUri struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

type CreateUserBody struct {
	Name     string `json:"name"     binding:"required,min=1,max=100"`
	Email    string `json:"email"    binding:"required,email"`
	Role     string `json:"role"     binding:"required,oneof=admin guest manager"`
	Password string `json:"password" binding:"required,min=6,max=255"`
}

type UpdateUserUri struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

type UpdateUserBody struct {
	Name     string `json:"name"     binding:"required,min=1,max=100"`
	Email    string `json:"email"    binding:"required,email"`
	Role     string `json:"role"     binding:"required,oneof=admin guest manager"`
	Password string `json:"password" binding:"omitempty,min=6,max=255"`
}

type DeleteUserUri struct {
	ID uint `uri:"id" binding:"required,min=1"`
}
