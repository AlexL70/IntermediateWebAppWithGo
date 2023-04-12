package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DBModel is the type for database connection values
type DBModel struct {
	DB *sql.DB
}

// Models is the wrapper for all models
type Models struct {
	DB DBModel
}

// NewModels returns a model type with database connection pool
func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBModel{DB: db},
	}
}

// Widget is the type for all widgets
type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

// Order is the type for all orders
type Order struct {
	ID            int       `json:"id"`
	WidgetID      int       `json:"widget_id"`
	TransactionID int       `json:"transaction_id"`
	StatusID      int       `json:"status_id"`
	Quantity      int       `json:"quantity"`
	Amount        int       `json:"amount"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

// Status is a type for order statuses
type Status struct {
	ID        int       `json:"id"`
	Name      int       `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TransactionStatus is a type for transaction statuses
type TransactionStatus struct {
	ID        int       `json:"id"`
	Name      int       `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Transactionis a type for transactions
type Transaction struct {
	ID                  int       `json:"id"`
	Amount              int       `json:"amount"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	BankReturnCode      string    `json:"bank_return_code"`
	TransactionStatusID int       `json:"transaction_status_id"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (m *DBModel) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var widget Widget

	stmt := `
		SELECT  id, name, description, inventory_level, price, coalesce(image, ''), created_at, updated_at
		  FROM  widgets
		 WHERE  id = ?
	`
	row := m.DB.QueryRowContext(ctx, stmt, id)
	err := row.Scan(&widget.ID, &widget.Name, &widget.Description, &widget.InventoryLevel,
		&widget.Price, &widget.Image, &widget.CreatedAt, &widget.UpdatedAt)
	if err != nil {
		return widget, fmt.Errorf("error reading widget from DB: %w", err)
	}
	return widget, nil
}

// InsertTransaction inserts new transaction and returns it's id
func (m *DBModel) InsertTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		insert into transactions (amount, currency, last_four, bank_return_code,
			transaction_status_id, created_at, updated_at)	
		values(?,?,?,?,?,?,?)
	`
	result, err := m.DB.ExecContext(ctx, stmt, txn.Amount, txn.Currency, txn.LastFour, txn.BankReturnCode,
		txn.TransactionStatusID, time.Now(), time.Now())
	if err != nil {
		return 0, fmt.Errorf("error adding transaction: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting transaction's last inserted id: %w", err)
	}
	return int(id), nil
}

// InsertOrder inserts new order and returns it's id
func (m *DBModel) InsertOrder(order Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		insert into orders (widget_id, transaction_id, status_id, quantity,
			amount, created_at, updated_at)	
		values(?,?,?,?,?,?,?)
	`
	result, err := m.DB.ExecContext(ctx, stmt, order.WidgetID, order.TransactionID, order.StatusID, order.Quantity,
		order.Amount, time.Now(), time.Now())
	if err != nil {
		return 0, fmt.Errorf("error adding order: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("error getting order's last inserted id: %w", err)
	}
	return int(id), nil
}
