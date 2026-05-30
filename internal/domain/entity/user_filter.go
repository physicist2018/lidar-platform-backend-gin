package entity

type UserFilter struct {
	Page  int
	Limit int
	Sort  string
	Role  UserRole
	Name  string
	Email string
}
