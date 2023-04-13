package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
	"github.com/go-chi/chi/v5"
)

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "terminal", td, "stripe-js"); err != nil {
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

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: paymentCurrency,
	}

	pi, err := card.RetrievePaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear
	amountInt, _ := strconv.Atoi(paymentAmount)
	amountStr := fmt.Sprintf("$%2.f", float32(amountInt)/100)

	data := map[string]any{
		"cardHolder":       cardHolder,
		"email":            email,
		"pi":               paymentIntent,
		"pm":               paymentMethod,
		"pa":               amountStr,
		"pc":               paymentCurrency,
		"last_four":        lastFour,
		"expiry_month":     expiryMonth,
		"expiry_year":      expiryYear,
		"bank_return_code": pi.LatestCharge.ID,
	}

	if err := app.renderTemplate(w, r, "succeeded", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
		return
	}
}

// ChargeOnce displays the page to buy one widget
func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)
	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	td := &templateData{
		Data: map[string]any{"widget": widget},
	}
	if err := app.renderTemplate(w, r, "buy-once", td, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}
