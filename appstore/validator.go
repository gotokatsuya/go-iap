package appstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/parnurzeal/gorequest"
)

const (
	SandboxURL    string = "https://sandbox.itunes.apple.com/verifyReceipt"
	ProductionURL string = "https://buy.itunes.apple.com/verifyReceipt"
)

// Config is a configuration to initialize client
type Config struct {
	IsProduction bool
	TimeOut      time.Duration
}

// IAPClient is an interface to call validation API in App Store
type IAPClient interface {
	Verify(IAPRequest) (*Receipt, error)
}

// Client implements IAPClient
type Client struct {
	URL     string
	TimeOut time.Duration
}

// HandleError returns error message by status code
func HandleError(status int) error {
	var message string

	switch status {
	case 0:
		return nil
	case 21006:
		return nil
	case 21000:
		message = "The App Store could not read the JSON object you provided."
	case 21002:
		message = "The data in the receipt-data property was malformed or missing."
	case 21003:
		message = "The receipt could not be authenticated."
	case 21004:
		message = "The shared secret you provided does not match the shared secret on file for your account."
	case 21005:
		message = "The receipt server is not currently available."
	case 21007:
		message = "This receipt is from the test environment, but it was sent to the production environment for verification. Send it to the test environment instead."
	case 21008:
		message = "This receipt is from the production environment, but it was sent to the test environment for verification. Send it to the production environment instead."
	default:
		message = "An unknown error ocurred"
	}

	return errors.New(message)
}

// New creates a client object
func New() Client {
	client := Client{
		URL:     SandboxURL,
		TimeOut: time.Second * 5,
	}
	if os.Getenv("IAP_ENVIRONMENT") == "production" {
		client.URL = ProductionURL
	}
	return client
}

// NewWithConfig creates a client with configuration
func NewWithConfig(config Config) Client {
	if config.TimeOut == 0 {
		config.TimeOut = time.Second * 5
	}

	client := Client{
		URL:     SandboxURL,
		TimeOut: config.TimeOut,
	}
	if config.IsProduction {
		client.URL = ProductionURL
	}

	return client
}

// Verify sends receipts and gets validation result
func (c *Client) Verify(req IAPRequest) (*Receipt, error) {
	res, body, errs := gorequest.New().
		Post(c.URL).
		Send(req).
		Timeout(c.TimeOut).
		End()

	if errs != nil {
		return nil, fmt.Errorf("%v", errs)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New("An error occurred in IAP - code:" + strconv.Itoa(res.StatusCode))
	}

	// iOS7 formant
	result := IAPResponseIOS7{
		rawReceipt: req.ReceiptData,
	}
	err := json.NewDecoder(strings.NewReader(body)).Decode(&result)
	if err != nil {
		return nil, err
	}
	if result.Environment != "" {
		return result.ToReceipt(), nil
	}

	// iOS6 formant
	resultIOS6 := IAPResponseIOS6{
		rawReceipt: req.ReceiptData,
	}
	err = json.NewDecoder(strings.NewReader(body)).Decode(&resultIOS6)
	if err != nil {
		return nil, err
	}
	return resultIOS6.ToIOS7().ToReceipt(), nil
}
