package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

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
	var payload = struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{true, err.Error()}

	return app.MarshalAndSendBack(w, payload)
}

func (app *application) MarshalAndSendBack(w http.ResponseWriter, data any) error {
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
	return nil
}
