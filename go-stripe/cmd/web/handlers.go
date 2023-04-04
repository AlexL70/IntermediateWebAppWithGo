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
