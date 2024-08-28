package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"strconv"

	"github.com/mvavassori/bare-analytics/models"
	"github.com/mvavassori/bare-analytics/services"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"

	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/webhook"
)

func init() {
	// todo: change it to prod key
	stripe.Key = "sk_test_51ONYpVEjL7fX4p99WhzOhVfRqbdGmvYlI37v6tkSThMAYJZJ5CVIhZSU6UWzVCH1AyIMk8ocxp1A56fFrNSSjzXn00JAfKJEsm"
}

func CreateCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("CreateCheckoutSession called")

		var req struct {
			Email       string `json:"email"`
			UserID      int    `json:"userId"`
			Plan        string `json:"plan"`
			PlanPriceID string `json:"-"`
		}

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		switch req.Plan {
		case "basic":
			req.PlanPriceID = "price_1Ppu8VEjL7fX4p99LqYqruOC"
		case "business":
			req.PlanPriceID = "price_1Ppu8VEjL7fX4p99LqYqruOC"
		default:
			http.Error(w, "Invalid plan", http.StatusBadRequest)
			return
		}

		// check if user exists
		user, err := services.GetUserById(db, req.UserID)
		if err != nil {
			log.Printf("models.GetUserById: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}

		var stripeCustomerID string
		if user.StripeCustomerID.Valid {
			stripeCustomerID = user.StripeCustomerID.String
		} else {
			customerParams := &stripe.CustomerParams{
				Email: stripe.String(req.Email),
			}
			customerParams.AddMetadata("userId", strconv.Itoa(req.UserID))
			customerParams.AddMetadata("originalEmail", req.Email)

			newCustomer, err := customer.New(customerParams)
			if err != nil {
				log.Printf("customer.New: %v", err)
				http.Error(w, "Error creating customer", http.StatusInternalServerError)
				return
			}
			stripeCustomerID = newCustomer.ID

			// Update user with new Stripe customer ID
			user.StripeCustomerID = sql.NullString{String: stripeCustomerID, Valid: true}
			err = services.AddStripeCustomerID(db, user)
			if err != nil {
				log.Printf("services.UpdateUser: %v", err)
				http.Error(w, "Error updating user", http.StatusInternalServerError)
				return
			}
		}

		params := &stripe.CheckoutSessionParams{
			Customer: &stripeCustomerID,
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(req.PlanPriceID),
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
			Metadata: map[string]string{
				"userId": strconv.Itoa(req.UserID),
				"plan":   req.Plan,
			},
			SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
				Metadata: map[string]string{
					"userId": strconv.Itoa(req.UserID),
					"plan":   req.Plan,
				},
			},
		}

		s, err := session.New(params)
		if err != nil {
			log.Printf("session.New: %v", err)
			http.Error(w, "Error creating checkout session", http.StatusInternalServerError)
			return
		}

		response := map[string]string{"url": s.URL}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Printf("json.NewEncoder: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
	}
}

// Pasted from stripe docs
func StripeWebhook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("StripeWebhook called")
		const MaxBodyBytes = int64(65536)
		bodyReader := http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		payload, err := io.ReadAll(bodyReader)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Replace this endpoint secret with your endpoint's unique secret
		// If you are testing with the CLI, find the secret by running 'stripe listen' // todo: change this.
		// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
		// at https://dashboard.stripe.com/webhooks
		endpointSecret := "whsec_1fc72b97ed963737df1dd4cc7ca20aac4989bf73120c87546a3e6abd2adf6fc0"
		signatureHeader := r.Header.Get("Stripe-Signature")

		event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Webhook signature verification failed. %v\n", err)
			w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
			return
		}
		// Unmarshal the event data into an appropriate struct depending on its Type
		switch event.Type {
		case "checkout.session.completed":
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			log.Printf("Checkout session completed for %s.", session.ID)

			// Access metadata from the session
			userId, ok := session.Metadata["userId"]
			if !ok {
				log.Printf("User ID not found in session metadata")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			log.Printf("User ID: %s", userId)

			plan, ok := session.Metadata["plan"]
			if !ok {
				log.Printf("Plan not found in session metadata")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			log.Printf("Plan: %s", plan)
			// get the user from the database
			var user models.User
			userIdInt, err := strconv.Atoi(userId)
			if err != nil {
				log.Printf("Error converting userId to int: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			user, err = services.GetUserById(db, userIdInt)
			if err != nil {
				log.Printf("Error getting user by ID: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			user.SubscriptionStatus = "active"
			user.SubscriptionPlan = sql.NullString{String: plan, Valid: true}
			// update the user's subscription status and plan in the database
			err = services.UpdateSubscriptionStatusAndPlan(db, user)
			if err != nil {
				log.Printf("Error updating subscription status: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// todo: send email to user

		case "customer.subscription.deleted":
			var subscription stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			log.Printf("Subscription deleted for %s.", subscription.ID)

			// get the user id from the subscription
			userId := subscription.Metadata["userId"]
			log.Printf("User ID: %s", userId)
			// get the user from the database
			var user models.User
			userIdInt, err := strconv.Atoi(userId)
			if err != nil {
				log.Printf("Error converting userId to int: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			user, err = services.GetUserById(db, userIdInt)
			if err != nil {
				log.Printf("Error getting user by ID: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			user.SubscriptionStatus = "inactive" // todo: figure out how to differentiate between canceled and not renewed
			user.SubscriptionPlan = sql.NullString{String: "", Valid: false}
			// update the user's subscription status and plan in the database
			err = services.UpdateSubscriptionStatusAndPlan(db, user)
			if err != nil {
				log.Printf("Error updating subscription status: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		// Then define and call a func to handle the deleted subscription.
		// handleSubscriptionCanceled(subscription)
		// case "customer.subscription.updated":
		// 	var subscription stripe.Subscription
		// 	err := json.Unmarshal(event.Data.Raw, &subscription)
		// 	if err != nil {
		// 		fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		return
		// 	}
		// 	log.Printf("Subscription updated for %s.", subscription.ID)
		// // Then define and call a func to handle the successful attachment of a PaymentMethod.
		// // handleSubscriptionUpdated(subscription)
		// case "customer.subscription.created":
		// 	var subscription stripe.Subscription
		// 	err := json.Unmarshal(event.Data.Raw, &subscription)
		// 	if err != nil {
		// 		fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		return
		// 	}
		// 	log.Printf("Subscription created for %s.", subscription.ID)
		// // Then define and call a func to handle the successful attachment of a PaymentMethod.
		// customerID := subscription.Customer.ID
		// log.Printf("Customer ID: %s", customerID)
		// // handleSubscriptionCreated(subscription)
		// case "customer.subscription.trial_will_end":
		// 	var subscription stripe.Subscription
		// 	err := json.Unmarshal(event.Data.Raw, &subscription)
		// 	if err != nil {
		// 		fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		return
		// 	}
		// 	log.Printf("Subscription trial will end for %s.", subscription.ID)
		// // Then define and call a func to handle the successful attachment of a PaymentMethod.
		// // handleSubscriptionTrialWillEnd(subscription)
		// case "entitlements.active_entitlement_summary.updated":
		// 	var subscription stripe.Subscription
		// 	err := json.Unmarshal(event.Data.Raw, &subscription)
		// 	if err != nil {
		// 		fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
		// 		w.WriteHeader(http.StatusBadRequest)
		// 		return
		// 	}
		// 	log.Printf("Active entitlement summary updated for %s.", subscription.ID)
		// // Then define and call a func to handle active entitlement summary updated.
		// // handleEntitlementUpdated(subscription)
		default:
			fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
		}
		w.WriteHeader(http.StatusOK)
	}
}
