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

type PasswordChange struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (u *User) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID                 int         `json:"id"`
		Name               string      `json:"name"`
		Email              string      `json:"email"`
		Websites           []Website   `json:"websites"`
		CreatedAt          time.Time   `json:"created_at"`
		UpdatedAt          time.Time   `json:"updated_at"`
		Role               string      `json:"role"`
		StripeCustomerID   interface{} `json:"stripe_customer_id,omitempty"`
		SubscriptionStatus string      `json:"subscription_status"`
		SubscriptionPlan   interface{} `json:"subscription_plan,omitempty"`
	}{
		ID:                 u.ID,
		Name:               u.Name,
		Email:              u.Email,
		Websites:           u.Websites,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
		Role:               u.Role,
		StripeCustomerID:   utils.NullableStringToJSON(u.StripeCustomerID),
		SubscriptionStatus: u.SubscriptionStatus,
		SubscriptionPlan:   utils.NullableStringToJSON(u.SubscriptionPlan),
	})
}

func (u *UserInsert) ValidateSignUp() error {
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
	if len(u.Password) < 6 { // todo change this to 8 before prod
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

func (pc *PasswordChange) ValidatePasswordChange() error {
	if pc.OldPassword == "" {
		return errors.New("old password is required")
	}
	if pc.NewPassword == "" {
		return errors.New("new password is required")
	}
	if pc.OldPassword == pc.NewPassword {
		return errors.New("new password must be different from old password")
	}
	if len(pc.NewPassword) < 8 {
		return errors.New("new password must be at least 8 characters long")
	}
	if !utils.HasSpecialChar(pc.NewPassword) || !utils.HasNumber(pc.NewPassword) || !utils.HasUppercase(pc.NewPassword) {
		return errors.New("new password must contain at least one uppercase letter, one number, and one special character")
	}
	return nil
}

func (uu *UserUpdate) ValidateUserUpdate() error {
	if uu.Name == "" {
		return errors.New("name is required")
	}
	if len(uu.Name) < 2 || len(uu.Name) > 50 {
		return errors.New("name must be between 2 and 50 characters")
	}
	if uu.Email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(uu.Email); err != nil {
		return errors.New("invalid email format")
	}
	return nil
}
