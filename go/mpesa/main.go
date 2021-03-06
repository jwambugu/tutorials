package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// B2CRequestBody is the body with the parameters to be used to initiate a B2C request
type B2CRequestBody struct {
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             string `json:"Amount"`
	PartyA             string `json:"PartyA"`
	PartyB             string `json:"PartyB"`
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occassion          string `json:"Occassion"`
}

// B2CRequestResponse is the response sent back after initiating a B2C request.
type B2CRequestResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
	RequestID                string `json:"requestId"`
	ErrorCode                string `json:"errorCode"`
	ErrorMessage             string `json:"errorMessage"`
}

// B2CCallbackResponse has the results of the callback data sent once we successfully make a B2C request.
type B2CCallbackResponse struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
		ResultParameters         struct {
			ResultParameter []struct {
				Key   string      `json:"Key"`
				Value interface{} `json:"Value"`
			} `json:"ResultParameter"`
		} `json:"ResultParameters"`
		ReferenceData struct {
			ReferenceItem struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"ReferenceItem"`
		} `json:"ReferenceData"`
	} `json:"Result"`
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

// setupHttpRequestWithAuth is a helper method aimed to create a http request adding
// the Authorization Bearer header with the access token for the Mpesa app.
func (m *Mpesa) setupHttpRequestWithAuth(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	accessTokenResponse, err := m.generateAccessToken()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessTokenResponse.AccessToken))

	return req, nil
}

// InitiateSTKPushRequest makes a http request performing an STK push request
func (m *Mpesa) InitiateSTKPushRequest(body *STKPushRequestBody) (*STKPushRequestResponse, error) {
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

	b2cRequestCallbackHandler := func(w http.ResponseWriter, req *http.Request) {
		payload := new(B2CCallbackResponse)

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("%+v\n", payload)

		fmt.Printf("Result Code: %d\n", payload.Result.ResultCode)
		fmt.Printf("Result Description: %s\n", payload.Result.ResultDesc)
	}

	addr := ":8080"
	http.HandleFunc("/stk-push-callback", stkPushCallbackHandler)
	http.HandleFunc("/b2c-callback", b2cRequestCallbackHandler)

	log.Printf("[*] Server started and running on port %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// GenerateSecurityCredentials returns the encrypted password using the public key of the specified environment
func GenerateSecurityCredentials(password string, isOnProduction bool) (string, error) {
	path := "./certificates/production.cer"

	if !isOnProduction {
		path = "./certificates/sandbox.cer"
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	contents, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(contents)

	var cert *x509.Certificate

	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}

	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	reader := rand.Reader

	encryptedPayload, err := rsa.EncryptPKCS1v15(reader, rsaPublicKey, []byte(password))
	if err != nil {
		return "", err
	}

	securityCredentials := base64.StdEncoding.EncodeToString(encryptedPayload)
	return securityCredentials, nil
}

// InitiateB2CRequest makes a http request performing a B2C payment request.
func (m *Mpesa) InitiateB2CRequest(body *B2CRequestBody) (*B2CRequestResponse, error) {
	url := fmt.Sprintf("%s/mpesa/b2c/v1/paymentrequest", m.baseURL)

	requestBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := m.setupHttpRequestWithAuth(http.MethodPost, url, requestBody)
	if err != nil {
		return nil, err
	}

	resp, err := m.makeRequest(req)
	if err != nil {
		return nil, err
	}

	b2cResponse := new(B2CRequestResponse)
	if err := json.Unmarshal(resp, &b2cResponse); err != nil {
		return nil, err
	}

	return b2cResponse, nil
}

func main() {
	httpServer()
}
