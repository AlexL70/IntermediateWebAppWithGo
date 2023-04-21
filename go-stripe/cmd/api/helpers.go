package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// errJsonPayload is for returning error/success information to client
type errJsonPayload = struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

// writeJson writes arbitrary data out (to response writer) as json
func (app *application) writeJson(w http.ResponseWriter, status int, data any, headers ...http.Header) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	if len(headers) > 0 {
		for k, v := range headers[0] {
			w.Header()[k] = v
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(out)
	return nil
}

// readJSON reads json request body into data. It only accepts single json value in the body
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1048576

	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return fmt.Errorf("error decoding request body: %w", err)
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("error decoding request body: it must only have a single JSON value")
	}

	return nil
}

func (app *application) BadRequest(w http.ResponseWriter, r *http.Request, err error) error {
	payload := errJsonPayload{
		Error:   true,
		Message: err.Error(),
	}

	return app.writeJson(w, http.StatusBadRequest, payload)
}

func (app *application) invalidCredentials(w http.ResponseWriter) error {
	payload := errJsonPayload{
		Error:   true,
		Message: "authentication failed; check your credentials and try again",
	}

	return app.writeJson(w, http.StatusUnauthorized, payload)
}
