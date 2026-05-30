package dto

type UserResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type UserPaginatedResponse struct {
	Data       []UserResponse `json:"data"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalItems int64          `json:"total_items"`
	TotalPages int            `json:"total_pages"`
}
