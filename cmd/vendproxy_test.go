package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/shophumm/humm-nz-vend/internal/pkg/config"
	"github.com/shophumm/humm-nz-vend/internal/pkg/humm"
	"github.com/shophumm/humm-nz-vend/internal/pkg/terminal"
	"github.com/shophumm/humm-nz-vend/internal/pkg/vend"

	shortid "github.com/ventu-io/go-shortid"
)

var Db *sql.DB

var appConfig config.HostConfig

func TestMain(m *testing.M) {
	appConfig, _ := config.ReadApplicationConfig("../configs/vendproxy.json")
	// we need a"" database connection for most of the tests

	Db = connectToDatabase(appConfig.Database)

	defer Db.Close()

	// add vars to the seesion to simulate a redirect
	DbSessionStore = initSessionStore(Db, appConfig.Session)

	terminal.Db = Db

	// humm.GatewayURL = "https://testpos.humm.com.au/webapi/v1/Test"

	returnCode := m.Run()

	os.Exit(returnCode)
}

// TestTerminalSave tests saving a new terminal in the database for the registration phase
func TestTerminalSave(t *testing.T) {
	// { Success SCRK01 Success VK5NGgc7nFJp 481f1e4098465f5229b33d91e0687c6123b91078e5c727b6d8ebf9360af145e7}
	var uniqueID, _ = shortid.Generate()

	terminal := &terminal.Terminal{
		FxlDeviceSigningKey: "VK5NGgc7nFJp",
		FxlRegisterID:       "Oxipos",
		FxlSellerID:         "30188105",
		Origin:              "http://pos.example.com",
		VendRegisterID:      uniqueID,
	}
	saved, err := terminal.Save("unit-test")

	if err != nil || saved == false {
		t.Fatal(err)
	}
}

// TestTerminalUniqueSave ensures that we get an error if we try to save the same terminal twice
func TestTerminalUniqueSave(t *testing.T) {
	// { Success SCRK01 Success VK5NGgc7nFJp 481f1e4098465f5229b33d91e0687c6123b91078e5c727b6d8ebf9360af145e7}

	terminal := &terminal.Terminal{
		FxlDeviceSigningKey: "VK5NGgc7nFJp",
		FxlRegisterID:       "Oxipos",
		FxlSellerID:         "30188105",
		Origin:              "http://pos.humm.com.au",
		VendRegisterID:      "0d33b6af-7d33-4913-a310-7cd187ad4756",
	}
	// insert the same record twice so that we know it's erroring
	saved, err := terminal.Save("unit-test")
	saved, err = terminal.Save("unit-test")

	if err != nil && saved != false {
		t.Fatal(err)

	}
}

// TestRegisterHandler  generating humm payload

func TestRegisterHandler(t *testing.T) {

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	form := url.Values{}
	form.Add("MerchantID", "30188105")
	form.Add("DeviceToken", "01SUCCES") // for this to work against sandbox or prod it needs a real token

	req, err := http.NewRequest(http.MethodPost, "/register", strings.NewReader(form.Encode()))

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	guid, _ := uuid.NewV4()
	vReq := &vend.PaymentRequest{
		RegisterID: guid.String(),
		Origin:     "http://testpos.humm.com.au",
	}
	rr := httptest.NewRecorder()

	session, err := getSession(req, "humm")

	session.Values["vReq"] = vReq
	err = session.Save(req, rr)

	if err != nil {
		log.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.

	handler := http.HandlerFunc(RegisterHandler)

	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d",
			status, http.StatusOK)
	}
}

// TestGeneratePayload generating humm payload assumes a registered device
// with both Humm and the local database
func TestProcessAuthorisationHandler(t *testing.T) {

	var uniqueID, _ = uuid.NewV4()
	log.Printf("Generated RegisterID of %s \n", uniqueID)
	terminal := &terminal.Terminal{
		FxlDeviceSigningKey: "1234567890", // use hardcoded signing key for dummy endpoint
		FxlRegisterID:       "Oxipos",
		FxlSellerID:         "30188105",
		Origin:              "http://pos.example.com",
		VendRegisterID:      uniqueID.String(),
	}

	// we do this to ensure that it's registered already,
	// otherwise we are going to get a 302
	saved, err := terminal.Save("unit-test")
	if saved != true {
		t.Error("Unable to save register")
		return
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	form := url.Values{}
	form.Add("amount", "4400")
	form.Add("origin", "http://pos.example.com")
	form.Add("paymentcode", "01APPROV") // needs a real payment code to succeed against sandbox / prod
	form.Add("register_id", uniqueID.String())

	req, err := http.NewRequest(http.MethodPost, "/pay", strings.NewReader(form.Encode()))

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PaymentHandler)

	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d",
			status, http.StatusOK)
		return
	}

	// Check the response body is what we expect.

	response := new(Response)
	body, _ := ioutil.ReadAll(rr.Body)
	err = json.Unmarshal(body, response)

	if response.Status != "ACCEPTED" {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "ACCEPTED")
	}
}

func TestProcessAuthorisationRedirect(t *testing.T) {

	var uniqueID, _ = uuid.NewV4()

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	form := url.Values{}
	form.Add("amount", "4400")
	form.Add("origin", "http://nonexistent.humm.com.au")
	form.Add("paymentcode", "012344")
	form.Add("register_id", uniqueID.String())

	req, err := http.NewRequest(http.MethodPost, "/pay", strings.NewReader(form.Encode()))

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(PaymentHandler)

	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusFound {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusFound)
	}

	if location := rr.HeaderMap.Get("Location"); location != "/register" {
		t.Errorf("Function redirects but redirects to %s rather than /register", location)
	}
}

func TestProcessSalesAdjustmentHandler(t *testing.T) {

	var uniqueID, _ = uuid.NewV4()
	log.Printf("Generated Refund TxID of : %s \n", uniqueID)

	vendRegisterID := "0afa8de1-1442-11e8-edec-94863fd13a3c"
	origin := "https://amtest.vendhq.com"

	// establish the session and save the amount and the register in the session
	vReq := &vend.PaymentRequest{
		Amount:     "4401",
		Origin:     origin,
		RegisterID: vendRegisterID,
	}

	// posTerminal, err := terminal.GetRegisteredTerminal("http://pos.example.com", vendRegisterID)

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	form := url.Values{}
	form.Add("purchasenumber", "01APPROV") // needs a real payment code to succeed against sandbox / prod

	req, err := http.NewRequest(http.MethodPost, "/refund", strings.NewReader(form.Encode()))
	session, err := getSession(req, "humm")

	if err != nil {
		t.Error("Can't get the session")
	}

	session.Values["vReq"] = vReq

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(RefundHandler)

	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d",
			status, http.StatusOK)
		return
	}

	// Check the response body is what we expect.
	response := new(Response)
	body, _ := ioutil.ReadAll(rr.Body)
	err = json.Unmarshal(body, response)

	if response.Status != "ACCEPTED" {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), "ACCEPTED")
	}
}

func TestProcessAuthorisationResponse(t *testing.T) {

	terminal, err := terminal.GetRegisteredTerminal("https://amtest.vendhq.com", "0afa8de1-1442-11e8-edec-94863fd13a3c")
	if err != nil {
		t.Error(err)
	}

	reponse := `{"x_purchase_number":"52011913","x_status":"Success","x_code":"SPRA01","x_message":"Approved","signature":"3b715be8fdd67decd299cbb14ceeec3c76667d48e3468e4d3f343602d9b7d690","tracking_data":null}`

	hummResponse := new(humm.HummResponse)
	err = json.Unmarshal([]byte(reponse), hummResponse)
	if err != nil {
		t.Error(err)
	}
	isValid := hummResponse.Authenticate(terminal.FxlDeviceSigningKey)

	if isValid == false {
		t.Error("Not a valid request")
	}
	browserResponse := processHummResponse(hummResponse, humm.Authorisation, "4000")

	if browserResponse.Status != statusAccepted {
		t.Error("Expecting for the transaction to be accepted")
	}
}

func TestRegistrationResponse(t *testing.T) {
	rawResponse := `{
		"x_key": "1234567890",
		"x_status": "Success",
		"x_code": "SCRK01",
		"x_message": "Success",
		"signature": "ff05ed059e8008a3e8e1210faee30ce0064e492f34709151804a32725d8441db",
		"tracking_data": null
	 }`

	hummResponse := new(humm.HummResponse)
	err := json.Unmarshal([]byte(rawResponse), hummResponse)
	if err != nil {
		t.Error(err)
	}
	isValid := hummResponse.Authenticate("Voh4ig3eepeedai8")

	if isValid == false {
		t.Error("Not a valid request")
	}
	browserResponse := processHummResponse(hummResponse, humm.Registration, "4000")

	if browserResponse.Status != statusAccepted {
		t.Error("Expecting for the transaction to be accepted")
	}

}

func TestGeneratePayload(t *testing.T) {

	log.Print("hello")
	hummPayload := humm.HummPayload{
		DeviceID:        "foobar",
		MerchantID:      "3342342",
		FinanceAmount:   "1000",
		FirmwareVersion: "version 4.0",
		OperatorID:      "John",
		PurchaseAmount:  "1000",
		PreApprovalCode: "1234",
	}

	var plainText = humm.GeneratePlainTextSignature(hummPayload)
	t.Log("Plaintext", plainText)

	signature := humm.SignMessage(plainText, "TEST")
	correctSig := "7dfd655530d41cee284b3e4cb7d08a058edf7b5641dffd15fdf1b61ff6a8699b"

	if signature != correctSig {
		t.Fatalf("expected %s but got %s", correctSig, signature)
	}
}
