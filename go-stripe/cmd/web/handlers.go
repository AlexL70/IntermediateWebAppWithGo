package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/models"
	"github.com/go-chi/chi/v5"
)

// VirtualTerminal displays virtual terminal page
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "terminal", td, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

// Home displays the home page
func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "home", td); err != nil {
		app.errorLog.Println(err)
	}
}

// PaymentSucceeded displays payment succeeded page
func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// read posted data
	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	email := r.Form.Get("cardholder_email")
	paymentIntent := r.Form.Get(("payment_intent"))
	paymentMethod := r.Form.Get(("payment_method"))
	paymentAmount := r.Form.Get(("payment_amount"))
	paymentCurrency := r.Form.Get(("payment_currency"))
	widgetID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.errorLog.Println(fmt.Errorf("Error converting widget id: %w", err))
		return
	}

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
	amountInt, err := strconv.Atoi(paymentAmount)
	if err != nil {
		app.errorLog.Println(fmt.Errorf("Error converting amount: %w", err))
		return
	}
	amountStr := fmt.Sprintf("$%2.f", float32(amountInt)/100)

	// create a new customer
	customerID, err := app.SaveCustomer(firstName, lastName, email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create a new transaction
	txn := models.Transaction{
		Amount:              amountInt,
		Currency:            paymentCurrency,
		LastFour:            lastFour,
		ExpiryMonth:         int(expiryMonth),
		ExpiryYear:          int(expiryYear),
		BankReturnCode:      pi.LatestCharge.ID,
		TransactionStatusID: 2, // Cleared
	}
	txnID, err := app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create a new order
	order := models.Order{
		WidgetID:      widgetID,
		TransactionID: txnID,
		CustomerID:    customerID,
		StatusID:      1, // Cleared
		Quantity:      1,
		Amount:        amountInt,
		DBEntity: models.DBEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	_, err = app.SaveOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := map[string]any{
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

	// should write this data to session and then redirect user to the new page

	if err := app.renderTemplate(w, r, "succeeded", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
		return
	}
}

func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}
	return app.DB.InsertCustomer(customer)
}

func (app *application) SaveTransaction(txn models.Transaction) (int, error) {
	return app.DB.InsertTransaction(txn)
}

func (app *application) SaveOrder(order models.Order) (int, error) {
	return app.DB.InsertOrder(order)
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
