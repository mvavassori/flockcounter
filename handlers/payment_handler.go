package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
)

func init() {
	stripe.Key = "sk_test_51ONYpVEjL7fX4p99WhzOhVfRqbdGmvYlI37v6tkSThMAYJZJ5CVIhZSU6UWzVCH1AyIMk8ocxp1A56fFrNSSjzXn00JAfKJEsm"
}

func CreateCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("CreateCheckoutSession")

		var req struct {
			Email  string `json:"email"`
			UserID string `json:"userId"`
			Plan   string `json:"plan"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Plan == "basic" {
			req.Plan = "price_1Ppu8VEjL7fX4p99LqYqruOC"
		}

		customerParams := &stripe.CustomerParams{
			Email: stripe.String(req.Email),
		}
		customerParams.AddMetadata("FinalEmail", req.Email)

		newCustomer, err := customer.New(customerParams)
		if err != nil {
			log.Printf("customer.New: %v", err)
			http.Error(w, "Error creating customer", http.StatusInternalServerError)
			return
		}

		params := &stripe.CheckoutSessionParams{
			Customer: &newCustomer.ID,
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(req.Plan),
					Quantity: stripe.Int64(1),
				},
			},
			Mode:         stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			SuccessURL:   stripe.String("http://localhost:3000/success"),
			CancelURL:    stripe.String("http://localhost:3000/canceled"),
			AutomaticTax: &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)},
			CustomerUpdate: &stripe.CheckoutSessionCustomerUpdateParams{
				Address: stripe.String("auto"),
			},
		}

		s, err := session.New(params)
		if err != nil {
			log.Printf("session.New: %v", err)
			http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, s.URL, http.StatusSeeOther)
	}
}
