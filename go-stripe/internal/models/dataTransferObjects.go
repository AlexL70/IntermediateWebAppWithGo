package models

type VTTransactionData struct {
	PaymentAmount   int    `json:"amount"`
	PaymentCurrency string `json:"currency"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	PaymentIntent   string `json:"payment_intent"`
	PaymentMethod   string `json:"payment_method"`
	BankReturnCode  string `json:"bank_return_code"`
	ExpiryMonth     int    `json:"expiry_month"`
	ExpiryYear      int    `json:"expiry_year"`
	LastFour        string `json:"last_four"`
}
