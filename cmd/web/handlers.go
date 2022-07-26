package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"myapp/internal/cards"
	"myapp/internal/encryption"
	"myapp/internal/models"
	"myapp/internal/urlsigner"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

//Displays the home page
func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}

//Displays the virtual terminal page
func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "terminal", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}

//transaction type for transactionData
type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount   int
	PaymentCurrency string
	LastFour        string
	ExpiryMonth     int
	ExpiryYear      int
	BankReturnCode  string
}

//GetTransactionData get transaction data from post and stripe
func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var txnData TransactionData
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}
	//read posted data
	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	//cardHolder := r.Form.Get("cardholder_name")
	email := r.Form.Get("email")
	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	// create transaction
	amount, err := strconv.Atoi(paymentAmount)
	if err != nil {
		app.errorLog.Println(err)
	}

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}

	pi, err := card.RetrievePaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}
	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return txnData, err
	}
	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	txnData = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntent,
		PaymentMethodID: paymentMethod,
		PaymentAmount:   amount,
		PaymentCurrency: paymentCurrency,
		LastFour:        lastFour,
		ExpiryMonth:     int(expiryMonth),
		ExpiryYear:      int(expiryYear),
		BankReturnCode:  pi.Charges.Data[0].ID,
	}
	return txnData, nil
}

//Invoice describes the JSON payload sent to the microservices
type Invoice struct {
	ID        int       `json:"id"`
	Quantity  int       `json:"quantity"`
	Amount    int       `json:"amount"`
	Product   string    `json:"product"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// PaymentSucceeded display the receipt page
func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	//read posted data

	widgetID, err := strconv.Atoi(r.Form.Get("product_id"))
	if err != nil {
		app.errorLog.Println(err)
	}
	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	//create customer
	customerId, err := app.SaveCustomer(txnData.FirstName, txnData.LastName, txnData.Email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      txnData.BankReturnCode,
		TransactionStatusID: 2,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
	}

	txnId, err := app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create order

	order := models.Order{
		WidgetID:      widgetID,
		TransactionID: txnId,
		CustomerID:    customerId,
		StatusID:      1,
		Quantity:      1,
		Amount:        txnData.PaymentAmount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	orderID, err := app.SaveOrders(order)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	//call micro service
	inv := Invoice{
		ID:        orderID,
		Amount:    order.Amount,
		Product:   "Widget",
		Quantity:  order.Quantity,
		FirstName: txnData.FirstName,
		LastName:  txnData.LastName,
		Email:     txnData.Email,
		CreatedAt: time.Now(),
	}

	err = app.callInvoiceMicro(inv)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	//write this data to session, and then redirect user to new page
	app.Session.Put(r.Context(), "receipt", txnData)

	http.Redirect(w, r, "/receipt", http.StatusSeeOther)

}
func (app *application) callInvoiceMicro(inv Invoice) error {

	url := "http://localhost:5000/invoice/create-and-send"
	out, err := json.MarshalIndent(inv, "", "\t")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(out))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	app.infoLog.Println(resp.Body)

	return nil

}

// VirtualTerminalPaymentSucceeded display the receipt for the virtual terminal transaction page
func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {

	txnData, err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	txn := models.Transaction{
		Amount:              txnData.PaymentAmount,
		Currency:            txnData.PaymentCurrency,
		LastFour:            txnData.LastFour,
		ExpiryMonth:         txnData.ExpiryMonth,
		ExpiryYear:          txnData.ExpiryYear,
		BankReturnCode:      txnData.BankReturnCode,
		TransactionStatusID: 2,
		PaymentIntent:       txnData.PaymentIntentID,
		PaymentMethod:       txnData.PaymentMethodID,
	}

	_, err = app.SaveTransaction(txn)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	//write this data to session, and then redirect user to new page
	app.Session.Put(r.Context(), "receipt", txnData)

	http.Redirect(w, r, "/virtual-terminal-receipt", http.StatusSeeOther)

}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "receipt", &templateDate{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "virtual-terminal-receipt", &templateDate{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

//SaveCustomer saves a customer and returns a id
func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	cus := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
	}
	id, err := app.DB.InsertCustomer(cus)
	if err != nil {
		return 0, err
	}
	return id, nil
}

//SaveTransaction saves a transaction and returns a id
func (app *application) SaveTransaction(transaction models.Transaction) (int, error) {

	id, err := app.DB.InsertTransaction(transaction)
	if err != nil {
		return 0, err
	}
	return id, nil
}

//SaveOrders saves a order and returns a id
func (app *application) SaveOrders(order models.Order) (int, error) {

	id, err := app.DB.InsertOrder(order)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// display the page to buy once
func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetId, err := strconv.Atoi(id)
	if err != nil {
		log.Println(err)
		app.errorLog.Println(err)
	}
	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		log.Println(err)
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget

	if err := app.renderTemplate(w, r, "buy-once", &templateDate{
		Data: data,
	}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

//BronzePlan displays the bronze paln page
func (app *application) BronzePlan(w http.ResponseWriter, r *http.Request) {
	widget, err := app.DB.GetWidget(2)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	data := make(map[string]interface{})
	data["widget"] = widget
	if err := app.renderTemplate(w, r, "bronze-plan", &templateDate{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

//BronzePlanReceipt displays the receipt for bronze plan
func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {

	if err := app.renderTemplate(w, r, "receipt-plan", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}

//LoginPage displays the login page.
func (app *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "login", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}

//PostLoginPage login securly
func (app *application) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	app.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	id, err := app.DB.Authenticate(email, password)
	if err != nil {
		app.errorLog.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.Session.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	app.Session.Destroy(r.Context())
	app.Session.RenewToken(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "forgot-password", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}
func (app *application) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	theURL := r.RequestURI

	testURL := fmt.Sprintf("%s%s", app.config.frontEnd, theURL)

	signer := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}

	valid := signer.VerifyToken(testURL)
	if !valid {
		app.errorLog.Println("Invalid URL - tampering detected")
		return
	}
	//make sure  not expired
	expired := signer.Expired(testURL, 60)
	if expired {
		app.errorLog.Println("Link Expired")
		return
	}

	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}

	encryptedEmail, err := encryptor.Encrypt(email)
	if err != nil {
		app.errorLog.Println("Encryption Failed")
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})

	data["email"] = encryptedEmail
	if err := app.renderTemplate(w, r, "reset-password", &templateDate{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

// AllSales display all one time purchsed orders
func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-sales", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}

}

//AllUsers shows all users page
func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-users", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}

}

//OneUser shows one admin user for add/edit/delete user
func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "one-user", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}

}

//AllSubscriptions display all subscriptions
func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "all-subscriptions", &templateDate{}); err != nil {
		app.errorLog.Println(err)
	}
}
func (app *application) ShowSale(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Sale"
	stringMap["cancel"] = "/admin/all-sales"
	stringMap["refund-url"] = "/api/admin/refund"
	stringMap["refund-btn"] = "Refund Order"
	stringMap["refunded-badge"] = "Refunded"
	stringMap["refunded-msg"] = "Charge Refunded"
	if err := app.renderTemplate(w, r, "sale", &templateDate{
		StringMap: stringMap,
	}); err != nil {
		app.errorLog.Println(err)
	}
}
func (app *application) ShowSubscription(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Subscription"
	stringMap["cancel"] = "/admin/all-subscriptions"
	stringMap["refund-url"] = "/api/admin/cancel-subscription"
	stringMap["refund-btn"] = "Cancel Subscription"
	stringMap["refunded-badge"] = "Cancelled"
	stringMap["refunded-msg"] = "Subscription Cancelled"
	if err := app.renderTemplate(w, r, "sale", &templateDate{
		StringMap: stringMap,
	}); err != nil {
		app.errorLog.Println(err)
	}
}
