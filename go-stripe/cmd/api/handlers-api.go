package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
)

type stripePayload struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
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
