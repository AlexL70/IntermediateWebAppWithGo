package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v74"
)

type stripePayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	CardBrand     string `json:"card_brand"`
	ExpiryMonth   int    `json:"exp_month"`
	ExpiryYear    int    `json:"exp_year"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

type jsonResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      int    `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload stripePayload

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	success := true

	pi, msg, err := card.Charge(payload.Currency, amount)
	if err != nil {
		success = false
	}

	var out []byte
	if success {
		out, err = json.MarshalIndent(pi, "", "  ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
	} else {
		j := jsonResponse{
			OK:      false,
			Message: msg,
		}

		out, err = json.MarshalIndent(j, "", "  ")
		if err != nil {
			app.errorLog.Println(err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) GetWidgetById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)
	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	out, err := json.MarshalIndent(widget, "", "  ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) CreateCustomerAndSubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		app.errorLog.Println(fmt.Errorf("error decoding data: %w", err))
		return
	}
	app.infoLog.Println(data.Email, data.LastFour, data.PaymentMethod, data.Plan)

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	var subscription *stripe.Subscription
	stripeCustomer, msg, err := card.CreateCustomer(data.PaymentMethod, data.Email)
	{
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
		subscription, err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email, data.LastFour, "")
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		} else {
			app.infoLog.Println("subscription id is", subscription.ID)
		}

		productID, err := strconv.Atoi(data.ProductID)
		if err != nil {
			err = fmt.Errorf("error converting ProductID to an int: %w", err)
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)
		if err != nil {
			err = fmt.Errorf("error saving customer: %w", err)
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
		amount, err := strconv.Atoi(data.Amount)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
		txn := models.Transaction{
			Amount:              amount,
			Currency:            "usd",
			LastFour:            data.LastFour,
			ExpiryMonth:         data.ExpiryMonth,
			ExpiryYear:          data.ExpiryYear,
			TransactionStatusID: 2,
		}
		txnID, err := app.SaveTransaction(txn)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
		order := models.Order{
			WidgetID:      productID,
			TransactionID: txnID,
			CustomerID:    customerID,
			StatusID:      1,
			Quantity:      1,
			Amount:        amount,
		}
		_, err = app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}
	}

FINISH:
	if okay {
		msg = "Transaction successful!"
	} else if !okay && msg == "" {
		msg = err.Error()
	}
	resp := jsonResponse{
		OK:      okay,
		Message: msg,
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		app.errorLog.Println("error marshalling response: %w", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
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
