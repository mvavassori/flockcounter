package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
)

// The init function is a special function in Go that runs automatically when the package is initialized
func init() {
	stripe.Key = "sk_test_51ONYpVEjL7fX4p99WhzOhVfRqbdGmvYlI37v6tkSThMAYJZJ5CVIhZSU6UWzVCH1AyIMk8ocxp1A56fFrNSSjzXn00JAfKJEsm"
}

func CreateCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// extract email from json body // todo: change it to model
		var req struct {
			Email string `json:"email"`
		}

		customerParams := &stripe.CustomerParams{
			Email: stripe.String(req.Email),
		}

		customerParams.AddMetadata("FinalEmail", req.Email)
		newCustomer, err := customer.New(customerParams)
		if err != nil {
			log.Printf("customer.New: %v", err)
		}

		params := &stripe.CheckoutSessionParams{
			Customer: &newCustomer.ID,
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				// &stripe.CheckoutSessionLineItemParams
				{
					// Provide the exact Price ID (for example, pr_1234) of the product you want to sell
					Price:    stripe.String("price_1Ppu8VEjL7fX4p99LqYqruOC"), // basic plan test
					Quantity: stripe.Int64(1),
				},
			},
			Mode:         stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			SuccessURL:   stripe.String("http://localhost:3000/success"),
			CancelURL:    stripe.String("http://localhost:3000/canceled"),
			AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
		}

		s, err := session.New(params)

		if err != nil {
			log.Printf("session.New: %v", err)
		}

		http.Redirect(w, r, s.URL, http.StatusSeeOther)
	}
}
