package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Mpesa is an application that will be making a transaction
type Mpesa struct {
	consumerKey    string
	consumerSecret string
	baseURL        string
	client         *http.Client
}

// MpesaOpts stores all the configuration keys we need to set up a Mpesa app,
type MpesaOpts struct {
	ConsumerKey    string
	ConsumerSecret string
	BaseURL        string
}

// MpesaAccessTokenResponse is the response sent back by Safaricom when we make a request to generate a token
type MpesaAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	RequestID    string `json:"requestId"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// STKPushRequestBody is the body with the parameters to be used to initiate an STK push request
type STKPushRequestBody struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            string `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushRequestResponse is the response sent back after initiating an STK push request.
type STKPushRequestResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
	RequestID           string `json:"requestId"`
	ErrorCode           string `json:"errorCode"`
	ErrorMessage        string `json:"errorMessage"`
}

// STKPushCallbackResponse has the results of the callback data sent once we successfully make an STK push request.
type STKPushCallbackResponse struct {
	Body struct {
		StkCallback struct {
			MerchantRequestID string `json:"MerchantRequestID"`
			CheckoutRequestID string `json:"CheckoutRequestID"`
			ResultCode        int    `json:"ResultCode"`
			ResultDesc        string `json:"ResultDesc"`
			CallbackMetadata  struct {
				Item []struct {
					Name  string      `json:"Name"`
					Value interface{} `json:"Value,omitempty"`
				} `json:"Item"`
			} `json:"CallbackMetadata"`
		} `json:"stkCallback"`
	} `json:"Body"`
}

// NewMpesa sets up and returns an instance of Mpesa
func NewMpesa(m *MpesaOpts) *Mpesa {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &Mpesa{
		consumerKey:    m.ConsumerKey,
		consumerSecret: m.ConsumerSecret,
		baseURL:        m.BaseURL,
		client:         client,
	}
}

// makeRequest performs all the http requests for the specific app
func (m *Mpesa) makeRequest(req *http.Request) ([]byte, error) {
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

// generateAccessToken sends a http request to generate new access token
func (m *Mpesa) generateAccessToken() (*MpesaAccessTokenResponse, error) {
	url := fmt.Sprintf("%s/oauth/v1/generate?grant_type=client_credentials", m.baseURL)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(m.consumerKey, m.consumerSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.makeRequest(req)
	if err != nil {
		return nil, err
	}

	accessTokenResponse := new(MpesaAccessTokenResponse)
	if err := json.Unmarshal(resp, &accessTokenResponse); err != nil {
		return nil, err
	}

	return accessTokenResponse, nil
}

// initiateSTKPushRequest makes a http request performing an STK push request
func (m *Mpesa) initiateSTKPushRequest(body *STKPushRequestBody) (*STKPushRequestResponse, error) {
	url := fmt.Sprintf("%s/mpesa/stkpush/v1/processrequest", m.baseURL)

	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	accessTokenResponse, err := m.generateAccessToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessTokenResponse.AccessToken))

	resp, err := m.makeRequest(req)
	if err != nil {
		return nil, err
	}

	stkPushResponse := new(STKPushRequestResponse)
	if err := json.Unmarshal(resp, &stkPushResponse); err != nil {
		return nil, err
	}

	return stkPushResponse, nil
}

func httpServer() {
	stkPushCallbackHandler := func(w http.ResponseWriter, req *http.Request) {
		payload := new(STKPushCallbackResponse)

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("%+v\n", payload)

		fmt.Printf("Result Code: %d\n", payload.Body.StkCallback.ResultCode)
		fmt.Printf("Result Description: %s\n", payload.Body.StkCallback.ResultDesc)
	}

	addr := ":8080"
	http.HandleFunc("/stk-push-callback", stkPushCallbackHandler)

	log.Printf("[*] Server started and running on port %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func main() {
	mpesa := NewMpesa(&MpesaOpts{
		ConsumerKey:    "Ybdrkh6fNDWjlicSZRDX2MReHqYSuZ4e",
		ConsumerSecret: "N0c8DTTOWeLLXqjm",
		BaseURL:        "https://sandbox.safaricom.co.ke",
	})

	// YYYYMMDDHHmmss
	timestamp := time.Now().Format("20060102150405")
	shortcode, passkey := "174379", "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"

	// base64 encoded Shortcode+Passkey+Timestamp
	passwordToEncode := fmt.Sprintf("%s%s%s", shortcode, passkey, timestamp)

	password := base64.StdEncoding.EncodeToString([]byte(passwordToEncode))

	response, err := mpesa.initiateSTKPushRequest(&STKPushRequestBody{
		BusinessShortCode: "1222",
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            "10",           // Amount to be charged when checking out
		PartyA:            "254708666389", // 2547XXXXXXXX
		PartyB:            shortcode,
		PhoneNumber:       "254708666389", // 2547XXXXXXXX
		CallBackURL:       "https://null.test",
		AccountReference:  "AABBCC",
		TransactionDesc:   "Payment via STK push.",
	})

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%+v\n", response)
}
