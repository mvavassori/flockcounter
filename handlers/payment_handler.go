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

	"github.com/mvavassori/bare-analytics/services"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/customer"
	"github.com/stripe/stripe-go/v79/webhook"
)

func init() {
	// change it to prod key
	stripe.Key = "sk_test_51ONYpVEjL7fX4p99WhzOhVfRqbdGmvYlI37v6tkSThMAYJZJ5CVIhZSU6UWzVCH1AyIMk8ocxp1A56fFrNSSjzXn00JAfKJEsm"
}

func CreateCheckoutSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("CreateCheckoutSession")

		var req struct {
			Email  string `json:"email"`
			UserID int    `json:"userId"`
			Plan   string `json:"plan"`
		}

		err := json.NewDecoder(r.Body).Decode(&req);
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Plan == "basic" {
			req.Plan = "price_1Ppu8VEjL7fX4p99LqYqruOC"
		} // todo add other plans

		// check if user exists
		user, err := services.GetUserById(db, req.UserID)
		if err != nil {
			log.Printf("models.GetUserByEmail: %v", err)
			http.Error(w, "Error getting user", http.StatusInternalServerError)
			return
		}


		var stripeCustomerID string
		if user.StripeCustomerID != "" {
			stripeCustomerID = user.StripeCustomerID
		} else {
			customerParams := &stripe.CustomerParams{
				Email: stripe.String(req.Email),
			}
			customerParams.AddMetadata("originalEmail", req.Email)
			customerParams.AddMetadata("userId", strconv.Itoa(req.UserID))

			newCustomer, err := customer.New(customerParams)
			if err != nil {
				log.Printf("customer.New: %v", err)
				http.Error(w, "Error creating customer", http.StatusInternalServerError)
				return
			}
			stripeCustomerID = newCustomer.ID

			// Update user with new Stripe customer ID
			user.StripeCustomerID = stripeCustomerID
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

		response := map[string]string{"url": s.URL}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Printf("json.NewEncoder: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}
}

// Pasted from stripe docs
// todo: add actions after cases
func StripeWebhook(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
    const MaxBodyBytes = int64(65536)
    bodyReader := http.MaxBytesReader(w, r.Body, MaxBodyBytes)
    payload, err := io.ReadAll(bodyReader)
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
      w.WriteHeader(http.StatusServiceUnavailable)
      return
    }
    // Replace this endpoint secret with your endpoint's unique secret
    // If you are testing with the CLI, find the secret by running 'stripe listen'
    // If you are using an endpoint defined with the API or dashboard, look in your webhook settings
    // at https://dashboard.stripe.com/webhooks
	// todo: change this. I got this from the cli for local testing with `stripe listen`.
    endpointSecret := "whsec_1fc72b97ed963737df1dd4cc7ca20aac4989bf73120c87546a3e6abd2adf6fc0"
    signatureHeader := r.Header.Get("Stripe-Signature")

	// // Log the payload and signature header for debugging
	// log.Printf("Payload: %s\n", string(payload))
	// log.Printf("Signature Header: %s\n", signatureHeader)

    event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
    if err != nil {
      fmt.Fprintf(os.Stderr, "⚠️  Webhook signature verification failed. %v\n", err)
      w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
      return
    }
    // Unmarshal the event data into an appropriate struct depending on its Type
    switch event.Type {
    case "customer.subscription.deleted":
      var subscription stripe.Subscription
      err := json.Unmarshal(event.Data.Raw, &subscription)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
      }
      log.Printf("Subscription deleted for %s.", subscription.ID)
      // Then define and call a func to handle the deleted subscription.
      // handleSubscriptionCanceled(subscription)
    case "customer.subscription.updated":
      var subscription stripe.Subscription
      err := json.Unmarshal(event.Data.Raw, &subscription)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
      }
      log.Printf("Subscription updated for %s.", subscription.ID)
      // Then define and call a func to handle the successful attachment of a PaymentMethod.
      // handleSubscriptionUpdated(subscription)
    case "customer.subscription.created":
      var subscription stripe.Subscription
      err := json.Unmarshal(event.Data.Raw, &subscription)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
      }
      log.Printf("Subscription created for %s.", subscription.ID)
      // Then define and call a func to handle the successful attachment of a PaymentMethod.
      // handleSubscriptionCreated(subscription)
    case "customer.subscription.trial_will_end":
      var subscription stripe.Subscription
      err := json.Unmarshal(event.Data.Raw, &subscription)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
      }
      log.Printf("Subscription trial will end for %s.", subscription.ID)
      // Then define and call a func to handle the successful attachment of a PaymentMethod.
      // handleSubscriptionTrialWillEnd(subscription)
    case "entitlements.active_entitlement_summary.updated":
      var subscription stripe.Subscription
      err := json.Unmarshal(event.Data.Raw, &subscription)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
      }
      log.Printf("Active entitlement summary updated for %s.", subscription.ID)
      // Then define and call a func to handle active entitlement summary updated.
      // handleEntitlementUpdated(subscription)
    default:
      fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
    }
    w.WriteHeader(http.StatusOK)
  }
}
