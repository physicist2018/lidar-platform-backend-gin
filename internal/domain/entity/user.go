package entity

import (
	"errors"
	"fmt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAdminRequired      = errors.New("admin role required")
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleGuest   UserRole = "guest"
	RoleManager UserRole = "manager"
)

func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleGuest, RoleManager:
		return true
	}
	return false
}

type User struct {
	ID       uint
	Name     string
	Email    string
	Role     UserRole
	Password string
}

// HidePassword blanks the password field for safe serialization (JSON/DTO).
func (u *User) HidePassword() {
	u.Password = ""
}

// ValidateRole ensures the role transition is allowed.
// Only admin can promote to admin/manager. Guests/managers cannot self-promote.
func (u *User) ValidateRole(requestedBy *User) error {
	if requestedBy == nil || requestedBy.Role != RoleAdmin {
		return fmt.Errorf("%w: only admin can change roles", ErrAdminRequired)
	}
	if !u.Role.IsValid() {
		return fmt.Errorf("invalid role: %s", u.Role)
	}
	return nil
}
