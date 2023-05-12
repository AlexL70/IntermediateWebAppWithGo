package main

import (
	"fmt"
	"net/http"
	"time"
)

// Order is the type for all orders
type Order struct {
	ID        int       `json:"id"`
	StatusID  int       `json:"status_id"`
	Quantity  int       `json:"quantity"`
	Amount    int       `json:"amount"`
	Product   string    `json:"product"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (app *application) CreateAndSendInvoice(w http.ResponseWriter, r *http.Request) {
	// receive json
	var order Order
	err := app.readJSON(w, r, &order)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	// generate a pdf invoice
	// create mail
	// send the mail with an attachment
	// send response
	resp := responsePayload{
		Error:   false,
		Message: fmt.Sprintf("Invoice %d.pdf has been created and sent to %s.", order.ID, order.Email),
	}
	err = app.writeJson(w, http.StatusOK, resp)
	if err != nil {
		app.errorLog.Println(err)
	}
}
