package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"time"
)

// stkPushExample is a sample of the M-Pesa Express (STK Push) request
func stkPushExample() {
	mpesa := NewMpesa(&MpesaOpts{
		ConsumerKey:    "your-consumer-key-goes-here",
		ConsumerSecret: "your-consumer-secret-goes-here",
		BaseURL:        "https://sandbox.safaricom.co.ke",
	})

	// The expected format is YYYYMMDDHHmmss
	timestamp := time.Now().Format("20060102150405")
	shortcode, passkey := "your-business-short-code-goes-here", "your-pass-key-goes-here"

	// base64 encoding of the shortcode + passkey + timestamp
	passwordToEncode := fmt.Sprintf("%s%s%s", shortcode, passkey, timestamp)
	password := base64.StdEncoding.EncodeToString([]byte(passwordToEncode))

	response, err := mpesa.InitiateSTKPushRequest(&STKPushRequestBody{
		BusinessShortCode: shortcode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            "10",                          // Amount to be charged when checking out
		PartyA:            "your-phone-number-goes-here", // 2547XXXXXXXX
		PartyB:            shortcode,
		PhoneNumber:       "your-phone-number-goes-here",              // 2547XXXXXXXX
		CallBackURL:       "your-endpoint-to-receive-the-callback-on", // https://
		AccountReference:  "TEST",
		TransactionDesc:   "Payment via STK push.",
	})

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%+v\n", response)
}

// b2cRequestExample is a sample of the B2C API request
func b2cRequestExample() {
	mpesa := NewMpesa(&MpesaOpts{
		ConsumerKey:    "your-consumer-key-goes-here",
		ConsumerSecret: "your-consumer-secret-goes-here",
		BaseURL:        "https://sandbox.safaricom.co.ke",
	})

	securityCredentials, err := GenerateSecurityCredentials("your-initiator-password", true)
	if err != nil {
		log.Fatalln(err)
	}

	response, err := mpesa.InitiateB2CRequest(&B2CRequestBody{
		InitiatorName:      "your-initiator-name-goes-here",
		SecurityCredential: securityCredentials,
		CommandID:          "BusinessPayment",
		Amount:             "1",
		PartyA:             "600983",
		PartyB:             "your-phone-number-goes-here",
		Remarks:            "Payment to customer",
		QueueTimeOutURL:    "your-endpoint-to-receive-notifications-in-case-request-times-out",
		ResultURL:          "your-endpoint-to-receive-the-notifications",
		Occassion:          "Payment to customer",
	})

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%+v\n", response)
}
