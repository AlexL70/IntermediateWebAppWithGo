package models

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DBModel is the type for database connection values
type DBModel struct {
	DB *gorm.DB
}

// Models is the wrapper for all models
type Models struct {
	DB DBModel
}

// NewModels returns a model type with database connection pool
func NewModels(db *gorm.DB) Models {
	return Models{
		DB: DBModel{DB: db},
	}
}

type IDBEntity interface {
	GetID() int
	SetCreated()
}
type DBEntity struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (e DBEntity) GetID() int {
	return e.ID
}

func (e *DBEntity) SetCreated() {
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
}

// Widget is the type for all widgets
type Widget struct {
	DBEntity
	Name           string `json:"name"`
	Description    string `json:"description"`
	InventoryLevel int    `json:"inventory_level"`
	Price          int    `json:"price"`
	Image          string `json:"image"`
	IsRecurring    bool   `json:"is_recurring"`
	PlanID         string `json:"plan_id"`
}

// Order is the type for all orders
type Order struct {
	DBEntity
	WidgetID      int         `json:"widget_id"`
	TransactionID int         `json:"transaction_id"`
	CustomerID    int         `json:"customer_id"`
	StatusID      int         `json:"status_id"`
	Quantity      int         `json:"quantity"`
	Amount        int         `json:"amount"`
	Widget        Widget      `json:"widget"`
	Transaction   Transaction `json:"transaction"`
	Customer      Customer    `json:"customer"`
}

// Status is a type for order statuses
type Status struct {
	DBEntity
	Name int `json:"name"`
}

// TransactionStatus is a type for transaction statuses
type TransactionStatus struct {
	DBEntity
	Name int `json:"name"`
}

// Transactionis a type for transactions
type Transaction struct {
	DBEntity
	Amount              int    `json:"amount"`
	Currency            string `json:"currency"`
	LastFour            string `json:"last_four"`
	ExpiryMonth         int    `json:"expiry_month"`
	ExpiryYear          int    `json:"expiry_year"`
	PaymentIntent       string `json:"payment_intent"`
	PaymentMethod       string `json:"payment_method"`
	BankReturnCode      string `json:"bank_return_code"`
	TransactionStatusID int    `json:"transaction_status_id"`
}

// User is a type for users
type User struct {
	DBEntity
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// Token is a type for saving tokens (SToken) to DB
type Token struct {
	DBEntity
	UserID    int
	Name      string
	Email     string
	Expiry    time.Time
	TokenHash []byte
}

// Customer is a type for customers
type Customer struct {
	DBEntity
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// GetWidget fetches Widget entity from DB by id
func (m *DBModel) GetWidget(id int) (Widget, error) {
	var widget Widget
	err := getEntityById(id, m, &widget)
	return widget, err
}

// GetTransaction fetches Transaction from DB by id
func (m *DBModel) GetTransaction(id int) (Transaction, error) {
	var tran Transaction
	err := getEntityById(id, m, &tran)
	return tran, err
}

// GetTransactionByPI fetches Transaction from DB by Payment Intent ID
func (m *DBModel) GetTransactionByPI(pi string) (Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx := m.DB.WithContext(ctx)

	var transaction Transaction
	if err := tx.First(&transaction, &Transaction{PaymentIntent: pi}).Error; err != nil {
		return transaction, fmt.Errorf("error reading Transaction from DB by payment intent id: %w", err)
	}

	return transaction, nil
}

// GetCustomer fetches Customer from DB by id
func (m *DBModel) GetCustomer(id int) (Customer, error) {
	var customer Customer
	err := getEntityById(id, m, &customer)
	return customer, err
}

// GetOrder fetches Order entity from DB by id
func (m *DBModel) GetOrder(id int) (Order, error) {
	var order Order
	err := getEntityById(id, m, &order)
	if err != nil {
		return order, err
	}
	order.Widget, err = m.GetWidget(order.WidgetID)
	if err != nil {
		return order, err
	}
	order.Transaction, err = m.GetTransaction(order.TransactionID)
	if err != nil {
		return order, err
	}
	order.Customer, err = m.GetCustomer(order.CustomerID)
	return order, err
}

// InsertTransaction inserts new transaction and returns it's id
func (m *DBModel) InsertTransaction(txn Transaction) (int, error) {
	return insertEntity(&txn, m)
}

// InsertOrder inserts new order and returns it's id
func (m *DBModel) InsertOrder(order Order) (int, error) {
	return insertEntity(&order, m)
}

// InsertCustomer inserts new order and returns it's id
func (m *DBModel) InsertCustomer(customer Customer) (int, error) {
	return insertEntity(&customer, m)
}

func (m *DBModel) UpdatePasswordForUser(u User, hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx := m.DB.WithContext(ctx)
	u.Password = hash
	u.UpdatedAt = time.Now()
	err := tx.Save(&u).Error

	return err
}

func (m *DBModel) UpdateOrderStatus(id, statusID int) error {
	var order Order
	err := getEntityById(id, m, &order)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx := m.DB.WithContext(ctx)
	order.StatusID = statusID

	return tx.Save(order).Error
}

// GetUserByEmail gets a user by email address
func (m *DBModel) GetUserByEmail(email string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx := m.DB.WithContext(ctx)
	user := User{}
	result := tx.Where(&User{Email: strings.ToLower(email)}).First(&user)
	if result.Error != nil {
		return user, fmt.Errorf("error searching user by email: %w", result.Error)
	}
	return user, nil
}

func (m *DBModel) Authenticate(email, password string) (int, error) {
	u, err := m.GetUserByEmail(email)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return 0, fmt.Errorf("incorrect password %w", err)
		}
		return 0, fmt.Errorf("error comparing password: %w", err)
	}
	return u.ID, nil
}

func (m *DBModel) GetUserForToken(token string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx := m.DB.WithContext(ctx)
	var user User
	tokenHash := sha256.Sum256([]byte(token))
	err := tx.Joins("join tokens t on t.user_id = users.id").
		Where("t.token_hash = ? and t.expiry > ?", tokenHash[:], time.Now()).
		First(&user).Error
	if err != nil {
		return nil, fmt.Errorf("error getting user by token from DB: %w", err)
	}
	return &user, nil
}

func (m *DBModel) InsertToken(t *SToken, u User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx := m.DB.WithContext(ctx)

	// Delete all existing tokens for the user first
	err := tx.Where(&Token{UserID: u.ID}).Delete(&Token{}).Error
	if err != nil {
		return 0, fmt.Errorf("error deleting old tokens: %w", err)
	}

	token := Token{
		UserID:    u.ID,
		Name:      fmt.Sprintf("%s %s", u.FirstName, u.LastName),
		Email:     u.Email,
		Expiry:    t.Expiry,
		TokenHash: t.Hash,
	}

	return insertEntity(&token, m)
}

func (m *DBModel) GetAllOrders() ([]*Order, int, error) {
	return getOrdersByRecurring(m, false, math.MaxInt, 1)
}

func (m *DBModel) GetAllSubscriptions() ([]*Order, int, error) {
	return getOrdersByRecurring(m, true, math.MaxInt, 1)
}

// getOrdersByRecurring returns the slice of orders
func getOrdersByRecurring(m *DBModel, isRecurring bool, pageSize, page int) ([]*Order, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tx := m.DB.WithContext(ctx)

	offset := (page - 1) * pageSize

	var orders []*Order
	result := tx.
		// I wonder is there any more elegant way to add "order by" clause to query in GORM?
		Clauses(clause.OrderBy{Columns: []clause.OrderByColumn{
			{
				Column: clause.Column{Table: "orders", Name: "created_at"},
				Desc:   true,
			},
		}}).
		InnerJoins("Widget", tx.Where(&Widget{IsRecurring: isRecurring}, "is_recurring")).
		Joins("Transaction").Joins("Customer").
		Offset(offset).
		Limit(pageSize).
		Find(&orders)
	if result.Error != nil {
		return nil, 0, fmt.Errorf("error getting all orders from DB: %w", result.Error)
	}

	var count int64
	cntResult := tx.Model(&Order{}).
		InnerJoins("Widget", tx.Where(&Widget{IsRecurring: isRecurring}, "is_recurring")).
		Count(&count)
	if cntResult.Error != nil {
		return nil, 0, fmt.Errorf("error getting orders' count from DB: %w", cntResult.Error)
	}

	return orders, int(count), nil
}

func insertEntity(entity IDBEntity, m *DBModel) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	entity.SetCreated()
	tx := m.DB.WithContext(ctx).Create(entity)
	if err := tx.Error; err != nil {
		return 0, fmt.Errorf("error adding %s: %w", reflect.TypeOf(entity), err)
	}

	return entity.GetID(), nil
}

func getEntityById(id int, m *DBModel, entity any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx := m.DB.WithContext(ctx)

	typeName := reflect.TypeOf(entity).String()
	if err := tx.First(entity, id).Error; err != nil {
		return fmt.Errorf("error reading %s from DB: %w", typeName, err)
	}

	return nil
}
