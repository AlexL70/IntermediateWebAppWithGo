package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/cards"
	common_models "github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/common"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/encryption"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/models"
	"github.com/AlexL70/IntermediateWebAppWithGo/go-stripe/internal/urlsigner"
	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v74"
	"golang.org/x/crypto/bcrypt"
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
			PaymentIntent:       subscription.ID,
			PaymentMethod:       data.PaymentMethod,
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
		orderID, err := app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			goto FINISH
		}

		inv := common_models.Order{
			ID:        orderID,
			Amount:    2000,
			Product:   "Bronze plan monthly subscription",
			Quantity:  order.Quantity,
			FirstName: data.FirstName,
			LastName:  data.LastName,
			Email:     data.Email,
			CreatedAt: time.Now(),
		}
		err = app.callInvoiceMicro(inv)
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

func (app *application) VitrualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	var txnData models.VTTransactionData
	err := app.readJSON(w, r, &txnData)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: "usd",
	}
	pi, err := card.RetrievePaymentIntent(txnData.PaymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	pm, err := card.GetPaymentMethod(txnData.PaymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
	}
	txnData.LastFour = pm.Card.Last4
	txnData.ExpiryMonth = int(pm.Card.ExpMonth)
	txnData.ExpiryYear = int(pm.Card.ExpYear)

	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      pi.LatestCharge.ID,
		TransactionStatusID: 2, // Cleared
		PaymentIntent:       txnData.PaymentIntent,
		PaymentMethod:       txnData.PaymentMethod,
	}

	txn.ID, err = app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	app.writeJson(w, http.StatusOK, txn)
}

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		mErr := app.BadRequest(w, r, err)
		if mErr != nil {
			app.errorLog.Println(mErr)
		}
		return
	}

	// get user from the database by email; send error if invalid email
	user, err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		app.errorLog.Println(err)
		app.invalidCredentials(w)
		return
	}
	app.infoLog.Println(user)

	// validate password; send error if invalid password
	passwordMatches, err := app.passwordMatches(user.Password, userInput.Password)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}
	if !passwordMatches {
		app.infoLog.Printf("Incorrect credentials entered by: %s\n", userInput.Email)
		app.invalidCredentials(w)
		return
	}

	// generate the token
	token, err := models.GenerateToken(user.ID, 12*time.Hour, models.ScopeAuthentication)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}

	// save token to the database
	_, err = app.DB.InsertToken(token, user)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}

	// send response
	payload := authJsonPayload{
		Error:   false,
		Message: fmt.Sprintf("Token for %q created.", userInput.Email),
		Token:   *token,
	}

	mErr := app.writeJson(w, http.StatusOK, payload)
	if mErr != nil {
		app.errorLog.Println(mErr)
	}
}

func (app *application) CheckAuthentication(w http.ResponseWriter, r *http.Request) {
	// validate the token and get associated user
	user, err := app.authenticateToken(r)
	//	sent back "invalid credentials" if token is not validated
	if err != nil {
		app.errorLog.Println(err)
		mErr := app.invalidCredentials(w)
		if mErr != nil {
			app.errorLog.Println(mErr)
		}
		return
	}
	// otherwise send back success response
	payload := responsePayload{
		Error:   false,
		Message: fmt.Sprintf("Authenticated user: %s", user.Email),
	}
	mErr := app.writeJson(w, http.StatusOK, payload)
	if mErr != nil {
		app.errorLog.Println(mErr)
	}
}

func (app *application) authenticateToken(r *http.Request) (*models.User, error) {
	// get and parse authorization header
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("authorization error; no authorization header in the request")
	}
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("bad authorization header's format")
	}
	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("wrong size of an authentication token")
	}
	// get the user from the tokens table
	user, err := app.DB.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no matching user found")
	}
	return user, nil
}

func (app *application) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	// prepare successful response
	response := struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{false, "If your email exists in our DB, then password reset link was successfully sent! Check your inbox!"}

	// verify that user with entered email exists in DB
	_, err = app.DB.GetUserByEmail(payload.Email)
	if err != nil {
		// print error to log because email is not found
		app.errorLog.Println(err)
		// return successful response so that hacker could not guess if email exists
		app.writeJson(w, http.StatusOK, response)
		return
	}

	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontEnd, payload.Email)
	sign := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}
	signedLink := sign.GenerateTokenFromString(link)

	var data = struct {
		Link string
	}{
		signedLink,
	}

	// send email
	err = app.SendMail("info@widget.com", payload.Email, "Password Reset Link", "password-reset", data)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}

	app.writeJson(w, http.StatusOK, response)
}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}
	decryptedEmail, err := encryptor.Decrypt(payload.Email)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, errors.New("wrong email data"))
	}

	user, err := app.DB.GetUserByEmail(decryptedEmail)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}

	err = app.DB.UpdatePasswordForUser(user, string(newHash))
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}

	response := struct {
		Error   bool   `json:"error"`
		Message string `json:"message"`
	}{false, fmt.Sprintf("Password has been successfully changed for user %q", decryptedEmail)}
	app.infoLog.Printf("Password has been changed for %q user", decryptedEmail)
	app.writeJson(w, http.StatusOK, response)
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	var pp paginationRequest
	err := app.readJSON(w, r, &pp)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, fmt.Errorf("incorrect pagination data; %w", err))
		return
	}
	allSales, count, err := app.DB.GetAllOrders(pp.PageSize, pp.CurrentPage)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}
	resp := paginatedResponse[models.Order]{
		paginationRequest: pp,
		LastPage:          lastPageNo(count, pp.PageSize),
		TotalRecords:      count,
		PageData:          allSales,
	}
	app.writeJson(w, http.StatusOK, resp)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	var pp paginationRequest
	err := app.readJSON(w, r, &pp)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, fmt.Errorf("incorrect pagination data; %w", err))
		return
	}
	allSubscriptions, count, err := app.DB.GetAllSubscriptions(pp.PageSize, pp.CurrentPage)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}
	resp := paginatedResponse[models.Order]{
		paginationRequest: pp,
		LastPage:          lastPageNo(count, pp.PageSize),
		TotalRecords:      count,
		PageData:          allSubscriptions,
	}
	app.writeJson(w, http.StatusOK, resp)
}

func (app *application) GetSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	orderId, err := strconv.Atoi(id)
	if err != nil {
		err := fmt.Errorf("error converting order id to int: %w", err)
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}
	order, err := app.DB.GetOrder(orderId)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	app.writeJson(w, http.StatusOK, order)
}

func (app *application) RefundCharge(w http.ResponseWriter, r *http.Request) {
	var chargeToRefund struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Amount        int    `json:"amount"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &chargeToRefund)
	if err != nil {
		err := fmt.Errorf("refund error: %w", err)
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)

	}

	// validate the amount and currency against transaction from DB
	trx, err := app.DB.GetTransactionByPI(chargeToRefund.PaymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}
	if trx.Amount != chargeToRefund.Amount {
		err := fmt.Errorf("refund error;wrong amount: %d; amount of transaction is %d", chargeToRefund.Amount, trx.Amount)
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	if trx.Currency != chargeToRefund.Currency {
		err := fmt.Errorf("refund error; wrong currency: %q; currency of transaction is %q", chargeToRefund.Currency, trx.Currency)
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: chargeToRefund.Currency,
	}

	err = card.Refund(chargeToRefund.PaymentIntent, chargeToRefund.Amount)

	if err != nil {
		app.errorLog.Println(err)
		var strErr *stripe.Error
		if ok := errors.As(err, &strErr); ok && strErr.Type == stripe.ErrorTypeInvalidRequest {
			app.BadRequest(w, r, errors.New(strErr.Msg))
		} else {
			app.internalError(w)
		}
		return
	}

	err = app.DB.UpdateOrderStatus(chargeToRefund.ID, 2)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, errors.New("the charge was refunded, but the database could not be updated; please call support"))
		return
	}

	resp := responsePayload{
		Error:   false,
		Message: "Refund succeeded!",
	}
	app.writeJson(w, http.StatusOK, resp)
}

func (app *application) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	var subToCancel struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"pi"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &subToCancel)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, fmt.Errorf("error cancelling subscription: %w", err))
		return
	}

	card := cards.Card{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: subToCancel.Currency,
	}

	err = card.CancelSubscription(subToCancel.PaymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		var strErr *stripe.Error
		if ok := errors.As(err, &strErr); ok && strErr.Type == stripe.ErrorTypeInvalidRequest {
			app.BadRequest(w, r, errors.New(strErr.Msg))
		} else {
			app.internalError(w)
		}
		return
	}

	err = app.DB.UpdateOrderStatus(subToCancel.ID, 3)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, errors.New("the subscription was cancelled, but the database could not be updated; please call support"))
		return
	}

	resp := responsePayload{
		Error:   false,
		Message: "Subscription cancelled",
	}
	app.writeJson(w, http.StatusOK, resp)
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	var pr paginationRequest
	err := app.readJSON(w, r, &pr)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, fmt.Errorf("incorrect pagination data; %w", err))
		return
	}
	allUsers, count, err := app.DB.GetAllUsers(pr.PageSize, pr.CurrentPage)
	if err != nil {
		app.errorLog.Println(err)
		app.internalError(w)
		return
	}
	resp := paginatedResponse[models.User]{
		paginationRequest: pr,
		LastPage:          lastPageNo(count, pr.PageSize),
		TotalRecords:      count,
		PageData:          allUsers,
	}
	app.writeJson(w, http.StatusOK, resp)
}

func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		e := fmt.Errorf("error converting user id to int: %w", err)
		app.errorLog.Println(e)
		app.BadRequest(w, r, e)
		return
	}
	user, err := app.DB.GetUserByID(userID)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, errors.New("user not found"))
		return
	}
	user.Password = ""
	app.writeJson(w, http.StatusOK, user)
}

func (app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		e := fmt.Errorf("error converting user id to int: %w", err)
		app.errorLog.Println(e)
		app.BadRequest(w, r, e)
		return
	}

	var user models.User
	err = app.readJSON(w, r, &user)
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	//	Save user
	var status int
	if userID > 0 { // update existing
		status = http.StatusOK
		user.ID = userID
		err = app.DB.UpdateUser(user)
	} else { // create new
		newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			app.errorLog.Println(err)
			app.BadRequest(w, r, errors.New("error updating password"))
			return
		}
		user.Password = string(newHash)
		user.ID, err = app.DB.InsertUser(user)
		status = http.StatusCreated
	}
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, err)
		return
	}

	//  Update password if necessary
	if userID > 0 && len(user.Password) > 0 {
		newHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			app.errorLog.Println(err)
			app.BadRequest(w, r, errors.New("error updating password"))
			return
		}
		err = app.DB.UpdatePasswordForUser(user, string(newHash))
		if err != nil {
			app.errorLog.Println(err)
			app.BadRequest(w, r, errors.New("error updating password"))
			return
		}
	}

	//	write response
	app.writeJson(w, status, responsePayload{
		Error:   false,
		Message: "User was successfully saved to the DB",
	})
}

func (app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(id)
	if err != nil {
		e := fmt.Errorf("error converting user id to int: %w", err)
		app.errorLog.Println(e)
		app.BadRequest(w, r, e)
		return
	}
	err = app.DB.DeleteUser(models.User{DBEntity: models.DBEntity{ID: userID}})
	if err != nil {
		app.errorLog.Println(err)
		app.BadRequest(w, r, errors.New("user not found"))
		return
	}
	app.writeJson(w, http.StatusOK, responsePayload{Error: false, Message: "User deleted successfully"})
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

func lastPageNo(recordCount, pageSize int) int {
	lastNo := recordCount / pageSize
	if recordCount%pageSize > 0 {
		lastNo++
	}
	return lastNo
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
