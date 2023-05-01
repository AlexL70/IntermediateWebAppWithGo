package cards

import (
	"fmt"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/paymentmethod"
	"github.com/stripe/stripe-go/v74/refund"
	"github.com/stripe/stripe-go/v74/subscription"
)

type Card struct {
	Secret   string
	Key      string
	Currency string
}

type Transaction struct {
	TransactionStatusID int
	Amount              int
	Currency            string
	LastFour            string
	BankReturnCode      string
}

func (c *Card) Charge(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	return c.createPaymentIntent(currency, amount)
}

func (c *Card) createPaymentIntent(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	stripe.Key = c.Secret

	// create a payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
	}

	// Here is how you can add some meta-data to transaction
	// params.AddMetadata("key", "value")

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg := fmt.Sprintf("%s – %s", stripeErr.Code, stripeErr.Msg)
			return nil, msg, err
		}
		return nil, "", err
	}
	return pi, "", nil
}

// GetPaymentMethod gets payment method by payment intent id
func (c *Card) GetPaymentMethod(s string) (*stripe.PaymentMethod, error) {
	stripe.Key = c.Secret

	pm, err := paymentmethod.Get(s, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting payment method: %w", err)
	}
	return pm, nil
}

// RetrievePaymentIntent returns existing PaymentIntent by id
func (c *Card) RetrievePaymentIntent(id string) (*stripe.PaymentIntent, error) {
	stripe.Key = c.Secret

	pi, err := paymentintent.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("error getting payment intent by id: %w", err)
	}
	return pi, nil
}

func (c *Card) SubscribeToPlan(cust *stripe.Customer, plan, email, last4, cardType string) (*stripe.Subscription, error) {
	stripeCustomerID := cust.ID
	items := []*stripe.SubscriptionItemsParams{
		{Plan: stripe.String(plan)},
	}
	params := &stripe.SubscriptionParams{
		Customer: stripe.String(stripeCustomerID),
		Items:    items,
	}
	params.AddMetadata("last_four", last4)
	params.AddMetadata("card_type", cardType)
	params.AddExpand("latest_invoice.payment_intent")
	subscription, err := subscription.New(params)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *Card) CreateCustomer(pm, email string) (*stripe.Customer, string, error) {
	stripe.Key = c.Secret
	custmerParams := &stripe.CustomerParams{
		PaymentMethod: stripe.String(pm),
		Email:         stripe.String(email),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm),
		},
	}
	cust, err := customer.New(custmerParams)
	if err != nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = fmt.Sprintf("%s – %s", stripeErr.Code, stripeErr.Msg)
		}
		return nil, msg, err
	}
	return cust, "", nil
}

func (c *Card) Refund(pi string, amount int) error {
	stripe.Key = c.Secret
	amountToRefund := int64(amount)
	refundParams := &stripe.RefundParams{
		Amount:        &amountToRefund,
		PaymentIntent: &pi,
	}

	_, err := refund.New(refundParams)
	if err != nil {
		return fmt.Errorf("failed refunding payment %q that sum is %d; error occured: %w", pi, amount, err)
	}

	return nil
}
