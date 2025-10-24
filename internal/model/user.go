package model

import "time"

type User struct {
	ID           string    `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	Role         string    `json:"role" db:"role"`
	Status       string    `json:"status" db:"status"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Attrs        any       `json:"attrs" db:"attrs"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
