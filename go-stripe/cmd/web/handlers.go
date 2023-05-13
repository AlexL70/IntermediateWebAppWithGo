package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
	common_models "github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/common"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/encryption"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/models"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/urlsigner"
	"github.com/go-chi/chi/v5"
)

// VirtualTerminal displays virtual terminal page
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "terminal", td); err != nil {
		app.errorLog.Println(err)
	}
}

// AllSales function displays a list of all sales
func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "all-sales", td); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "all-subscriptions", td); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ShowSale(w http.ResponseWriter, r *http.Request) {
	td := &templateData{StringMap: map[string]string{
		"title":           "Sale",
		"backUrl":         "/admin/all-sales",
		"backCaption":     "Back to all sales",
		"refund-url":      "/api/admin/refund",
		"refund-btn":      "Refund order",
		"refunded-msg":    "Charge refunded!",
		"refunded-status": "refunded",
	},
	}
	if err := app.renderTemplate(w, r, "sale", td); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ShowSubscription(w http.ResponseWriter, r *http.Request) {
	td := &templateData{StringMap: map[string]string{
		"title":           "Subscription",
		"backUrl":         "/admin/all-subscriptions",
		"backCaption":     "Back to all subscriptions",
		"refund-url":      "/api/admin/cancel-subscription",
		"refund-btn":      "Cancel subscription",
		"refunded-msg":    "Subscription cancelled!",
		"refunded-status": "cancelled",
	},
	}
	if err := app.renderTemplate(w, r, "sale", td); err != nil {
		app.errorLog.Println(err)
	}
}

// AllUsers shows list of all admin users
func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "all-users", td); err != nil {
		app.errorLog.Println(err)
	}
}

// OneUser shows one admin user for add/edit/delete
func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	td := &templateData{}
	if err := app.renderTemplate(w, r, "one-user", td); err != nil {
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

type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount   int
	PaymentCurrency string
	LastFour        string
	ExpiryMonth     int
	ExpiryYear      int
	BankReturnCode  string
}

// GetTransactionData gets transaction data from post and stripe
func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var txnData TransactionData
	err := r.ParseForm()
	if err != nil {
		return txnData, err
	}

	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
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
		return txnData, err
	}

	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		return txnData, err
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear
	amountInt, err := strconv.Atoi(paymentAmount)
	if err != nil {
		return txnData, fmt.Errorf("error converting amount: %w", err)
	}

	txnData = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntent,
		PaymentMethodID: paymentMethod,
		PaymentAmount:   amountInt,
		PaymentCurrency: paymentCurrency,
		LastFour:        lastFour,
		ExpiryMonth:     int(expiryMonth),
		ExpiryYear:      int(expiryYear),
		BankReturnCode:  pi.LatestCharge.ID,
	}
	return txnData, nil
}

// PaymentSucceeded displays payment succeeded page
func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widgetID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.errorLog.Println(fmt.Errorf("error converting widget id: %w", err))
		return
	}
	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.infoLog.Println(err)
		return
	}

	// create a new customer
	customerID, err := app.SaveCustomer(txnData.FirstName, txnData.LastName, txnData.Email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create a new transaction
	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
		BankReturnCode:      txnData.BankReturnCode,
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
		Amount:        txnData.PaymentAmount,
		DBEntity: models.DBEntity{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	orderID, err := app.SaveOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// call the microservice that forms invoice and send it to customer
	invoice := common_models.Order{
		ID:        orderID,
		Amount:    order.Amount,
		Product:   "Widget",
		Quantity:  order.Quantity,
		FirstName: txnData.FirstName,
		LastName:  txnData.LastName,
		Email:     txnData.Email,
		CreatedAt: time.Now(),
	}
	err = app.callInvoiceMicro(invoice)
	if err != nil {
		app.errorLog.Println(err)
		app.Session.Put(r.Context(), "error", fmt.Sprintf("Error generating invoice: %s", err))
	}

	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "/receipt", http.StatusSeeOther)
}

func (app *application) callInvoiceMicro(inv common_models.Order) error {
	url := "http://localhost:5000/invoice/create-and-send"
	out, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling invoice: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(out))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error calling invoice microservice: %w", err)
	}
	defer resp.Body.Close()
	app.infoLog.Println(resp.Body)
	return nil
}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Pop(r.Context(), "receipt").(TransactionData)
	data := map[string]any{
		"txn": txn,
	}

	if err := app.renderTemplate(w, r, "receipt", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
		return
	}
}

// VirtualTerminalPaymentSucceeded displays payment succeeded page for virtual terminal transactions
func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.infoLog.Println(err)
		return
	}

	// create a new transaction
	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
		BankReturnCode:      txnData.BankReturnCode,
		TransactionStatusID: 2, // Cleared
	}
	_, err = app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.Session.Put(r.Context(), "receipt", txnData)
	http.Redirect(w, r, "/virtual-terminal-receipt", http.StatusSeeOther)
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Pop(r.Context(), "receipt").(TransactionData)
	data := map[string]any{
		"txn": txn,
	}

	if err := app.renderTemplate(w, r, "virtual-terminal-receipt", &templateData{
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

func (app *application) BronzePlan(w http.ResponseWriter, r *http.Request) {
	widget, err := app.DB.GetWidget(2)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	data := map[string]any{
		"widget": widget,
	}
	if err := app.renderTemplate(w, r, "bronze-plan", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(fmt.Errorf("error rendering template: %w", err))
		return
	}
}

func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "receipt-bronze-plan", &templateData{}); err != nil {
		app.errorLog.Println(fmt.Errorf("error rendering template: %w", err))
		return
	}
}

// LoginPage shows the login page
func (app *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "login", &templateData{}); err != nil {
		app.errorLog.Println(fmt.Errorf("error rendering template: %w", err))
		return
	}
}

// LoginPage processes post (submit) of login form
func (app *application) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	id, err := app.DB.Authenticate(email, password)
	if err != nil {
		app.errorLog.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	app.Session.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	app.Session.Destroy(r.Context())
	app.Session.RenewToken(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "forgot-password", &templateData{}); err != nil {
		app.errorLog.Println(fmt.Errorf("error rendering template: %w", err))
		return
	}
}

func (app *application) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	testUrl := fmt.Sprintf("%s%s", app.config.frontEnd, r.RequestURI)
	signer := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}
	if valid := signer.VerifyToken(testUrl); !valid {
		app.errorLog.Printf("Invalid URL tampering detected: %q\n", testUrl)
		return
	}

	// make sure token has not expired yet
	expired := signer.Expired(testUrl, 60)
	if expired {
		app.errorLog.Println("Change password URL has been expired.")
		return
	}

	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}

	encryptedEmail, err := encryptor.Encrypt(email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := map[string]any{
		"email": encryptedEmail,
	}
	if err := app.renderTemplate(w, r, "reset-password", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(fmt.Errorf("error rendering template: %w", err))
		return
	}
}
