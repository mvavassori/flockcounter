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

	"github.com/mvavassori/flockcounter/models"
	"github.com/mvavassori/flockcounter/services"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"

	"github.com/stripe/stripe-go/v81/customer"
	"github.com/stripe/stripe-go/v81/webhook"
)

type PlanDetails struct {
	Plan     string
	Interval string
}

// todo change to prod price ids
var planToPriceID = map[PlanDetails]string{
	{"basic", "monthly"}:    "price_1Qh8V9EjL7fX4p99Boqi7jO9",
	{"basic", "yearly"}:     "price_1Qh8XNEjL7fX4p99pAeKMecA",
	{"business", "monthly"}: "price_1Qh8VSEjL7fX4p99nlMJ5yFB",
	{"business", "yearly"}:  "price_1Qh8WSEjL7fX4p99id1NdSfG",
}

var priceIDToPlan = make(map[string]PlanDetails)

func init() {
	// todo: change it to prod key
	stripe.Key = os.Getenv("STRIPE_KEY")

	// Populate the reverse map from planToPriceID // made this to being able define and update PriceID mappings in one place (planToPriceID)
	for planDetails, priceID := range planToPriceID { // iterate over the planToPriceId map and "grab" the planDetails key and the priceID values
		priceIDToPlan[priceID] = planDetails // create a map that maps the priceID (of the original planToPriceID map) to the planDetails. (basically invert the planToPriceID from key:value to value:key)
	}
}

func CreateCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email       string `json:"email"`
			UserID      int    `json:"userId"`
			Plan        string `json:"plan"`
			PlanPriceID string `json:"-"`
			Interval    string `json:"interval"`
		}
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// check if user exists
		user, err := services.GetUserById(db, req.UserID)
		if err != nil {
			log.Printf("models.GetUserById: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}
		// Check for existing subscriptions
		existingSubscriptions, err := services.GetActiveSubscription(user.StripeCustomerID.String)
		if err != nil {
			http.Error(w, "Unable to check subscriptions", http.StatusInternalServerError)
			return
		}

		// Prevent creation of a new session if an active subscription exists
		if existingSubscriptions != nil {
			http.Error(w, "User already has an active subscription", http.StatusConflict)
			return
		}

		planDetails := PlanDetails{Plan: req.Plan, Interval: req.Interval}
		priceID, found := planToPriceID[planDetails]
		if !found {
			http.Error(w, "Invalid plan or interval", http.StatusBadRequest)
			return
		}
		req.PlanPriceID = priceID

		var stripeCustomerID string
		if user.StripeCustomerID.Valid && user.StripeCustomerID.String != "" {
			stripeCustomerID = user.StripeCustomerID.String
		} else {
			customerParams := &stripe.CustomerParams{
				Email: stripe.String(req.Email),
			}
			customerParams.AddMetadata("userId", strconv.Itoa(req.UserID))
			customerParams.AddMetadata("originalEmail", req.Email)
			newCustomer, err := customer.New(customerParams)
			if err != nil {
				log.Printf("Failed to create customer: %v", err)
				http.Error(w, "Error creating customer", http.StatusInternalServerError)
				return
			}
			stripeCustomerID = newCustomer.ID
			_, err = db.Exec("UPDATE users SET stripe_customer_id = $1 WHERE id = $2",
				sql.NullString{String: stripeCustomerID, Valid: true}, user.ID)
			if err != nil {
				log.Printf("Failed to update user with customer ID: %v", err)
				http.Error(w, "Error updating user", http.StatusInternalServerError)
				return
			}
		}

		params := &stripe.CheckoutSessionParams{
			Customer: stripe.String(stripeCustomerID),
			LineItems: []*stripe.CheckoutSessionLineItemParams{
				{
					Price:    stripe.String(req.PlanPriceID),
					Quantity: stripe.Int64(1),
				},
			},
			Mode:         stripe.String(string(stripe.CheckoutSessionModeSubscription)),
			SuccessURL:   stripe.String("http://localhost:3000/success"),  // todo change to prod
			CancelURL:    stripe.String("http://localhost:3000/canceled"), // todo change to prod
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
			// recipientEmail := "famigliavavassori@outlook.it"
			// subject := "Subscription Success Subject"
			// body := "Your subscription has been successful."

			// utils.SendEmail(recipientEmail, subject, body)

		case "customer.subscription.deleted":
			var subscription stripe.Subscription
			err := json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			log.Printf("Subscription deleted for %s.", subscription.ID)

			// access metadata from the subscription
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
			user.SubscriptionStatus = "inactive"
			user.SubscriptionPlan = sql.NullString{String: "", Valid: false}
			// update the user's subscription status and plan in the database
			err = services.UpdateSubscriptionStatusAndPlan(db, user)
			if err != nil {
				log.Printf("Error updating subscription status: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// todo: send email to user
			// recipientEmail := "famigliavavassori@outlook.it"
			// subject := "Subscription Canceled Subject"
			// body := "Your subscription has been canceled."

			// utils.SendEmail(recipientEmail, subject, body)

			// Then define and call a func to handle the deleted subscription.
			// handleSubscriptionCanceled(subscription)

		case "customer.subscription.updated":
			var subscription stripe.Subscription
			err = json.Unmarshal(event.Data.Raw, &subscription)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			log.Printf("Subscription updated for %s.", subscription.ID)

			// Extracting user ID from metadata
			userId := subscription.Metadata["userId"]
			log.Printf("User ID: %s", userId)

			userIdInt, err := strconv.Atoi(userId)
			if err != nil {
				log.Printf("Error converting userId to int: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Get the user from the database
			user, err := services.GetUserById(db, userIdInt)
			if err != nil {
				log.Printf("Error getting user by ID: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Determine subscription status
			user.SubscriptionStatus = string(subscription.Status) // Subscription status directly from Stripe (e.g., active, past_due, canceled)

			priceID := subscription.Items.Data[0].Price.ID
			planDetails, found := priceIDToPlan[priceID] // The second argument in maps (in this case `found`) returns true if the key exists in the map otherwise false
			if found {
				user.SubscriptionPlan = sql.NullString{String: planDetails.Plan, Valid: true}
			} else {
				user.SubscriptionPlan = sql.NullString{Valid: false} // Set to NULL if not found
			}

			// Update the user's subscription status and plan in the database
			err = services.UpdateSubscriptionStatusAndPlan(db, user)
			if err != nil {
				log.Printf("Error updating subscription status: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

		// todo: send email to user
		// recipientEmail := "famigliavavassori@outlook.it"
		// subject := "Subscription Canceled Subject"
		// body := "Your subscription has been canceled."

		// utils.SendEmail(recipientEmail, subject, body)

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

// func TestEmailSending() http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		log.Println("TestEmailSending handler called")

// 		recipientEmail := "famigliavavassori@outlook.it"
// 		subject := "Test Email Subject"
// 		body := "This is a test email body sent from Go!"

// 		log.Printf("Sending test email to: %s", recipientEmail)
// 		err := utils.SendEmail(recipientEmail, subject, body)
// 		if err != nil {
// 			log.Printf("Failed to send test email: %v", err)
// 			w.WriteHeader(http.StatusInternalServerError)
// 			return
// 		}
// 		log.Println("Test email sent successfully!")
// 		w.WriteHeader(http.StatusOK)
// 	}
// }
