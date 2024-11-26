package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/mail"
	"time"

	"github.com/mvavassori/bare-analytics/utils"
)

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

type GetUserResponse struct {
	User                   User   `json:"user"`
	SubscriptionExpiryDate string `json:"subscriptionExpiryDate,omitempty"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	var stripeCustomerID string
	if u.StripeCustomerID.Valid {
		stripeCustomerID = u.StripeCustomerID.String
	}

	var subscriptionPlan string
	if u.SubscriptionPlan.Valid {
		subscriptionPlan = u.SubscriptionPlan.String
	}

	return json.Marshal(&struct {
		ID                 int       `json:"id"`
		Name               string    `json:"name"`
		Email              string    `json:"email"`
		Websites           []Website `json:"websites"`
		CreatedAt          time.Time `json:"created_at"`
		UpdatedAt          time.Time `json:"updated_at"`
		Role               string    `json:"role"`
		StripeCustomerID   string    `json:"stripe_customer_id,omitempty"`
		SubscriptionStatus string    `json:"subscription_status"`
		SubscriptionPlan   string    `json:"subscription_plan,omitempty"`
	}{
		ID:                 u.ID,
		Name:               u.Name,
		Email:              u.Email,
		Websites:           u.Websites,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
		Role:               u.Role,
		StripeCustomerID:   stripeCustomerID,
		SubscriptionStatus: u.SubscriptionStatus,
		SubscriptionPlan:   subscriptionPlan,
	})
}

func (u *UserInsert) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if len(u.Name) < 2 || len(u.Name) > 50 {
		return errors.New("name must be between 2 and 50 characters")
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
	if len(u.Password) < 6 { // todo change this to 8
		return errors.New("password must be at least 8 characters long")
	}
	if !utils.HasSpecialChar(u.Password) || !utils.HasNumber(u.Password) || !utils.HasUppercase(u.Password) {
		return errors.New("password must contain at least one uppercase letter, one number, and one special character")
	}
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
	return nil
}
