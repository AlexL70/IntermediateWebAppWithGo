package card

import (
	"fmt"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
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
