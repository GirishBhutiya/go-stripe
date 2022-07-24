package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//DBModel is the type for database connection values
type DBModel struct {
	DB *sql.DB
}

//Models is the wrapper for all models
type Models struct {
	DB DBModel
}

//NewModels returns a model type with database connection pool
func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBModel{
			DB: db,
		},
	}
}

//Widget is the tyoe for all widgets
type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	IsRecurring    bool      `json:"is_recurring"`
	PlanID         string    `json:"plan_id"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

//Order type for all orders
type Order struct {
	ID            int         `json:"id"`
	WidgetID      int         `json:"widget_id"`
	TransactionID int         `json:"transaction_id"`
	CustomerID    int         `json:"customer_id"`
	StatusID      int         `json:"status_id"`
	Quantity      int         `json:"quantity"`
	Amount        int         `json:"amount"`
	CreatedAt     time.Time   `json:"-"`
	UpdatedAt     time.Time   `json:"-"`
	Widget        Widget      `json:"widget"`
	Transaction   Transaction `json:"transaction"`
	Customer      Customer    `json:"customer"`
}

//Status type for all order status
type Status struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//transaction Status type for all transaction status
type TransactionStatus struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//transaction type for transaction
type Transaction struct {
	ID                  int       `json:"id"`
	Amount              int       `json:"amount"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	ExpiryMonth         int       `json:"expiry_month"`
	ExpiryYear          int       `json:"expiry_year"`
	PaymentIntent       string    `json:"payment_intent"`
	PaymentMethod       string    `json:"payment_method"`
	BankReturnCode      string    `json:"bank_return_code"`
	TransactionStatusID int       `json:"transaction_status_id"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

//Users is the type for all users
type Users struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Password  string    `json:"passwrd"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//Customer is the type for all Customers
type Customer struct {
	ID          int       `json:"id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Email       string    `json:"email"`
	ExpiryMonth string    `json:"expiry_month"`
	ExpiryYear  string    `json:"expiry_year"`
	CreatedAt   time.Time `json:"-"`
	UpdatedAt   time.Time `json:"-"`
}

//GetWidget get one widget by id
func (m *DBModel) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var widget Widget
	row := m.DB.QueryRowContext(ctx, `SELECT id, name ,description ,inventory_level ,price, COALESCE(image,''), is_recurring, plan_id, created_at, updated_at 
		FROM widgets where id=?`, id)

	err := row.Scan(&widget.ID,
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.IsRecurring,
		&widget.PlanID,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)
	if err != nil {
		return widget, err
	}
	return widget, nil
}

//InsertTransaction insert a new transaction and returns its id
func (m *DBModel) InsertTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO transactions
		(amount,currency, last_four, bank_return_code,transaction_status_id,expiry_month,expiry_year,payment_intent,payment_method,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`

	result, err := m.DB.ExecContext(ctx, stmt,
		txn.Amount,
		txn.Currency,
		txn.LastFour,
		txn.BankReturnCode,
		txn.TransactionStatusID,
		txn.ExpiryMonth,
		txn.ExpiryYear,
		txn.PaymentIntent,
		txn.PaymentMethod,
		time.Now(),
		time.Now())

	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

//InsertOrder insert a new order and returns its id
func (m *DBModel) InsertOrder(order Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO orders
		(widget_id,transaction_id, status_id, quantity,amount,customer_id,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?)`

	result, err := m.DB.ExecContext(ctx, stmt,
		order.WidgetID,
		order.TransactionID,
		order.StatusID,
		order.Quantity,
		order.Amount,
		order.CustomerID,
		time.Now(),
		time.Now())

	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

//InsertCustomer insert a new customer and returns its id
func (m *DBModel) InsertCustomer(customer Customer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO customers
		(first_name,last_name, email, created_at,updated_at)
		VALUES (?,?,?,?,?)`

	result, err := m.DB.ExecContext(ctx, stmt,
		customer.FirstName,
		customer.LastName,
		customer.Email,
		time.Now(),
		time.Now())

	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

//GetUserByEmail get a user by email address
func (m *DBModel) GetUserByEmail(email string) (Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email = strings.ToLower(email)

	var u Users
	stmt := `SELECT id,first_name,last_name,email,password,created_at,updated_at 
		FROM users 
		WHERE email=?`
	row := m.DB.QueryRowContext(ctx, stmt, email)

	err := row.Scan(&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Password,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (m *DBModel) Authenticate(email, password string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email = strings.ToLower(email)

	var id int
	var hashedPassword string

	stmt := `SELECT id,password FROM users WHERE email = ?`
	row := m.DB.QueryRowContext(ctx, stmt, email)

	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, errors.New("incorrect password")
	} else if err != nil {
		return 0, err
	}

	return id, nil
}
func (m *DBModel) UpdatePasswordForUSer(u Users, hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET password = ? WHERE id = ? `

	_, err := m.DB.ExecContext(ctx, stmt, hash, u.ID)
	if err != nil {
		return err
	}
	return nil
}
func (m *DBModel) GetAllOrders(orderType int) ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `select 
				o.id,o.widget_id,o.transaction_id,o.customer_id, o.status_id,o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name, t.id,t.amount ,t.currency , t.last_four ,t.expiry_month,
				t.expiry_year ,t.payment_intent ,t.bank_return_code , c.id,c.first_name ,c.last_name ,c.email 
			from 
				orders o
				left join widgets w on (o.widget_id = w.id)
				left join transactions t on (o.transaction_id = t.id)
				left join customers c on (o.customer_id = c.id)
			where 
				w.is_recurring = ?
			order by
				o.created_at desc`

	var orders []*Order

	rows, err := m.DB.QueryContext(ctx, stmt, orderType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

//GetAllOrdersPaginated returns a slice of a subset  of orders
func (m *DBModel) GetAllOrdersPaginated(pageSize, page, orderType int) ([]*Order, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	offset := (page - 1) * pageSize

	stmt := `select 
				o.id,o.widget_id,o.transaction_id,o.customer_id, o.status_id,o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name, t.id,t.amount ,t.currency , t.last_four ,t.expiry_month,
				t.expiry_year ,t.payment_intent ,t.bank_return_code , c.id,c.first_name ,c.last_name ,c.email 
			from 
				orders o
				left join widgets w on (o.widget_id = w.id)
				left join transactions t on (o.transaction_id = t.id)
				left join customers c on (o.customer_id = c.id)
			where 
				w.is_recurring = ?
			order by
				o.created_at desc
			limit ? offset ?`

	var orders []*Order

	rows, err := m.DB.QueryContext(ctx, stmt, orderType, pageSize, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, 0, 0, err
		}
		orders = append(orders, &o)
	}
	stmt = `SELECT count(o.id) FROM orders o LEFT JOIN widgets w on (o.widget_id = w.id) WHERE w.is_recurring = ?`

	var totalRecords int
	countRow := m.DB.QueryRowContext(ctx, stmt, orderType)
	err = countRow.Scan(&totalRecords)
	if err != nil {
		return nil, 0, 0, err
	}

	lastPage := totalRecords / pageSize

	return orders, lastPage, totalRecords, nil
}

func (m *DBModel) GetOrderById(orderId int) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `select 
				o.id,o.widget_id,o.transaction_id,o.customer_id, o.status_id,o.quantity, o.amount, o.created_at, o.updated_at, w.id, w.name, t.id,t.amount ,t.currency , t.last_four ,t.expiry_month,
				t.expiry_year ,t.payment_intent ,t.bank_return_code , c.id,c.first_name ,c.last_name ,c.email 
			from 
				orders o
				left join widgets w on (o.widget_id = w.id)
				left join transactions t on (o.transaction_id = t.id)
				left join customers c on (o.customer_id = c.id)
			where 
				o.id = ?`

	var o Order

	row := m.DB.QueryRowContext(ctx, stmt, orderId)

	err := row.Scan(
		&o.ID,
		&o.WidgetID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Widget.ID,
		&o.Widget.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.ExpiryMonth,
		&o.Transaction.ExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)
	if err != nil {
		return o, err
	}
	return o, nil
}

func (m *DBModel) UpdateOrderStatus(id, statusID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE orders set status_id = ? WHERE id = ?`

	_, err := m.DB.ExecContext(ctx, stmt, statusID, id)
	if err != nil {
		return err
	}
	return nil
}

func (m *DBModel) GetAllUsers() ([]*Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var users []*Users

	stmt := `SELECT id, last_name, first_name,email, created_at, updated_at
		FROM 
			users
		ORDER BY
			last_name,first_name`

	rows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u Users
		err = rows.Scan(
			&u.ID,
			&u.LastName,
			&u.FirstName,
			&u.Email,
			&u.CreatedAt,
			&u.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (m *DBModel) GetOneUSer(id int) (Users, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var u Users

	stmt := `SELECT id, last_name, first_name,email, created_at, updated_at
		FROM 
			users
		WHERE id = ?`

	row := m.DB.QueryRowContext(ctx, stmt, id)

	err := row.Scan(
		&u.ID,
		&u.LastName,
		&u.FirstName,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return u, err
	}

	return u, nil
}

func (m *DBModel) EditUser(u Users) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET 
		first_name = ?,
		last_name = ?,
		email = ?,
		updated_at = ?
	WHERE id = ?`

	_, err := m.DB.ExecContext(ctx, stmt, u.FirstName, u.LastName, u.Email, time.Now(), u.ID)
	if err != nil {
		return err
	}
	return nil
}
func (m *DBModel) AddUser(u Users, hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO users (first_name,last_name,email,password, created_at,updated_at)
		VALUES (?,?,?,?,?,?)`

	_, err := m.DB.ExecContext(ctx, stmt, u.FirstName, u.LastName, u.Email, hash, time.Now(), time.Now())
	if err != nil {
		return err
	}
	return nil
}

func (m *DBModel) DeleteUser(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `DELETE from users WHERE id = ?`

	_, err := m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	stmt = `DELETE from tokens WHERE user_id = ?`
	_, err = m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	return nil
}
