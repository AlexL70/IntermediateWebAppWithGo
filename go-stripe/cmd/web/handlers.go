package main

import "net/http"

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	td := &templateData{
		StringMap: map[string]string{
			"STRIPE_KEY": app.config.stripe.key,
		},
	}
	if err := app.renderTemplate(w, r, "terminal", td); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// read posted data
	cardHolder := r.Form.Get("cardholder_name")
	email := r.Form.Get("cardholder_email")
	paymentIntent := r.Form.Get(("payment_intent"))
	paymentMethod := r.Form.Get(("payment_method"))
	paymentAmount := r.Form.Get(("payment_amount"))
	paymentCurrency := r.Form.Get(("payment_currency"))

	data := map[string]any{
		"cardHolder": cardHolder,
		"email":      email,
		"pi":         paymentIntent,
		"pm":         paymentMethod,
		"pa":         paymentAmount,
		"pc":         paymentCurrency,
	}

	if err := app.renderTemplate(w, r, "succeeded", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
		return
	}
}
