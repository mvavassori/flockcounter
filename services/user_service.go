package services

import (
	"database/sql"

	"github.com/mvavassori/bare-analytics/models"
)

func GetUserById(db *sql.DB, id int) (models.User, error) {
	var user models.User
	err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt, &user.Role, &user.StripeCustomerID, &user.SubscriptionStatus, &user.SubscriptionPlan)
	if err != nil {
		return user, err
	}
	return user, nil
}

func AddStripeCustomerID(db *sql.DB, user models.User) error {
	_, err := db.Exec("UPDATE users SET stripe_customer_id = $1 WHERE id = $2", user.StripeCustomerID, user.ID)
	if err != nil {
		return err
	}
	return nil
}

func UpdateSubscriptionStatus(db *sql.DB, user models.User) error {
	_, err := db.Exec("UPDATE users SET subscription_status = $1 WHERE id = $2", user.SubscriptionStatus, user.ID)
	if err != nil {
		return err
	}
	return nil
}
