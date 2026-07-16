package user

import (
	"github.com/google/uuid"
)

type Role string

const (
	RoleGhost Role = ""
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	id           uuid.UUID
	email        string
	passwordHash string
	role         Role
}

func (u *User) ID() uuid.UUID {
	return u.id
}

func (u *User) Email() string {
	return u.email
}

func (u *User) PasswordHash() string {
	return u.passwordHash
}

func (u *User) Role() Role {
	return u.role
}
