package models

import (
	"errors"
	"net/mail"
)

type User struct {
	ID       int       `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Password string    `json:"password"` //``json:"-"` to hide the field
	Websites []Website `json:"websites"` // Slice of websites owned by the user
}

type UserInsert struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"` //``json:"-"`
}

type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"` //``json:"-"`
}

func (u *UserInsert) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return errors.New("invalid email format")
	}
	if u.Password == "" {
		return errors.New("password is required")
	}
	// Add more validation rules as needed
	return nil
}

func (u *UserLogin) ValidateLogin() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(u.Email); err != nil {
		return errors.New("invalid email format")
	}
	if u.Password == "" {
		return errors.New("password is required")
	}
	// Add more validation rules as needed
	return nil
}
