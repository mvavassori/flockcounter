package services

import (
	"database/sql"
	"fmt"

	// "encoding/json"
	// "log"

	"github.com/mvavassori/bare-analytics/models"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/subscription"
	// "github.com/stripe/stripe-go/v79/customer"
	// "github.com/stripe/stripe-go/v79/subscription"
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

func UpdateSubscriptionStatusAndPlan(db *sql.DB, user models.User) error {
	_, err := db.Exec("UPDATE users SET subscription_status = $1, subscription_plan = $2 WHERE id = $3", user.SubscriptionStatus, user.SubscriptionPlan, user.ID)
	if err != nil {
		return err
	}
	return nil
}

func GetActiveSubscriptions(customerID string) ([]*stripe.Subscription, error) {
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String("active"),
	}
	params.Filters.AddFilter("limit", "", "100") // Set a limit as needed

	iter := subscription.List(params)

	var activeSubscriptions []*stripe.Subscription

	for iter.Next() {
		subscription := iter.Subscription()
		activeSubscriptions = append(activeSubscriptions, subscription)
	}

	if err := iter.Err(); err != nil {
		return nil, err
	}

	fmt.Println("Active subscriptions:", activeSubscriptions)

	return activeSubscriptions, nil
}
