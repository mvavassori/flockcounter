package models

import (
	"database/sql"
	"errors"
	"net/mail"
	"time"
)

// todo:fix stuff here

type User struct {
	ID                 int            `json:"id"`
	Name               string         `json:"name"`
	Email              string         `json:"email"`
	Password           string         `json:"-"`        //``json:"-"` to hide the field
	Websites           []Website      `json:"websites"` // Slice of websites owned by the user
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	Role               string         `json:"role"`
	StripeCustomerID   sql.NullString `json:"stripe_customer_id"`
	SubscriptionStatus string         `json:"subscription_status"` // default value is "inactive"
	SubscriptionPlan   sql.NullString `json:"subscription_plan"`
}

type UserInsert struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserUpdate struct {
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserLogin struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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
