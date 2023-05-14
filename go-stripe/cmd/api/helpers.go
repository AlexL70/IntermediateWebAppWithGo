package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type responsePayload struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

type validationResponsePayload struct {
	responsePayload
	Errors map[string]string `json:"errors"`
}

// authJsonPayload is for returning error/success information to client
type authJsonPayload struct {
	Error   bool          `json:"error"`
	Message string        `json:"message"`
	Token   models.SToken `json:"authentication_token"`
}

type paginationRequest struct {
	PageSize    int `json:"page_size"`
	CurrentPage int `json:"current_page"`
}

type paginatedResponse[E any] struct {
	paginationRequest
	LastPage     int  `json:"last_page"`
	TotalRecords int  `json:"total_records"`
	PageData     []*E `json:"page_data"`
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
	payload := responsePayload{
		Error:   true,
		Message: err.Error(),
	}

	return app.writeJson(w, http.StatusBadRequest, payload)
}

func (app *application) invalidCredentials(w http.ResponseWriter) error {
	payload := responsePayload{
		Error:   true,
		Message: "authentication failed; check your credentials and try again",
	}

	return app.writeJson(w, http.StatusUnauthorized, payload)
}

func (app *application) internalError(w http.ResponseWriter) error {
	payload := responsePayload{
		Error:   true,
		Message: "internal server error; if it repeats, please contact the support",
	}
	return app.writeJson(w, http.StatusInternalServerError, payload)
}

func (app *application) passwordMatches(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, fmt.Errorf("error checking password: %w", err)
		}
	}
	return true, nil
}

func (app *application) failedValidation(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	payload := validationResponsePayload{
		responsePayload: responsePayload{Error: true, Message: "Validation failed!"},
		Errors:          errors,
	}
	app.writeJson(w, http.StatusUnprocessableEntity, payload)
}
