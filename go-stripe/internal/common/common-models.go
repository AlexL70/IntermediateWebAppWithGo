package common_models

import "time"

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
