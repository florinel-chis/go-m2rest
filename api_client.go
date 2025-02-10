package magento2

import (
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/pkg/errors"

	"gopkg.in/resty.v1"
)

const (
	RetryAttempts       = 3
	RetryWaitSeconds    = 5
	RetryMaxWaitSeconds = 20
)

var logger *log.Logger

func SetLogger(l *log.Logger) {
	logger = l
	if logger == nil {
		logger = log.Default() // Ensure logger is not nil
	}
	log.SetOutput(logger.Writer()) // Redirect default log output if needed.
	log.SetFlags(logger.Flags())
	log.SetPrefix(logger.Prefix())
}

func init() {
	// Initialize with a default logger if SetLogger is not called.
	if logger == nil {
		logger = log.Default()
	}
}

type Client struct {
	HTTPClient *resty.Client
}

type StoreConfig struct {
	Scheme    string
	HostName  string
	StoreCode string
}

func (c *Client) GetRouteAndDecode(route string, target interface{}, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return errors.WithStack(ErrNoPointer)
	}

	log.Debugf("GET request to: %s", route)
	resp, err := c.HTTPClient.R().SetResult(target).Get(route)
	if err != nil {
		log.Errorf("GET request to %s failed: %v", route, err)
	} else {
		log.Debugf("GET request to %s completed with status: %d", route, resp.StatusCode())
	}
	return mayReturnErrorForHTTPResponse(err, resp, tryTo)
}

func (c *Client) PostRouteAndDecode(route string, body, target interface{}, tryTo string) error {
	if reflect.TypeOf(target).Kind() != reflect.Ptr {
		return errors.WithStack(ErrNoPointer)
	}

	log.Debugf("POST request to: %s, body: %+v", route, body)
	resp, err := c.HTTPClient.R().SetResult(target).SetBody(body).Post(route)
	if err != nil {
		log.Errorf("POST request to %s failed: %v", route, err)
	} else {
		log.Debugf("POST request to %s completed with status: %d", route, resp.StatusCode())
	}
	return mayReturnErrorForHTTPResponse(err, resp, tryTo)
}

func NewAPIClientWithoutAuthentication(storeConfig *StoreConfig) *Client {
	httpClient := buildBasicHTTPClient(storeConfig)
	log.Infof("Created API client without authentication for store: %+v", storeConfig)

	return &Client{
		HTTPClient: httpClient,
	}
}

func NewAPIClientFromAuthentication(storeConfig *StoreConfig, payload AuthenticationRequestPayload, authenticationType AuthenticationType) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	log.Infof("Authenticating API client for store: %+v, authenticationType: %s, payload: %+v", storeConfig, authenticationType.Route(), payload)
	resp, err := client.R().SetBody(payload).Post(authenticationType.Route())
	if err != nil {
		return nil, err
	}

	token := mayTrimSurroundingQuotes(resp.String())
	client.SetAuthToken(token)
	log.Infof("API client authenticated successfully, token obtained. Authentication type: %s", authenticationType.Route())

	return &Client{
		HTTPClient: client,
	}, nil
}

func NewAPIClientFromIntegration(storeConfig *StoreConfig, bearer string) (*Client, error) {
	client := buildBasicHTTPClient(storeConfig)

	client.SetAuthToken(bearer)
	log.Infof("Created API client from integration for store: %+v, bearer token provided", storeConfig)

	return &Client{
		HTTPClient: client,
	}, nil
}

func buildBasicHTTPClient(storeConfig *StoreConfig) *resty.Client {
	apiVersion := "/V1"
	restPrefix := "/rest/" + storeConfig.StoreCode
	fullRestRoute := storeConfig.Scheme + "://" + storeConfig.HostName + restPrefix + apiVersion
	client := resty.New()
	client.SetRESTMode()
	client.SetHostURL(fullRestRoute)
	client.SetHeaders(map[string]string{
		"User-Agent": "go-m2rest",
	})
	client.SetDebug(false) // Set to true for very verbose resty debugging

	retryWait := time.Duration(RetryWaitSeconds)
	retryMaxWait := time.Duration(RetryMaxWaitSeconds)
	client.SetRetryCount(RetryAttempts).
		SetRetryWaitTime(retryWait * time.Second).
		SetRetryMaxWaitTime(retryMaxWait * time.Second).
		AddRetryCondition(
			func(r *resty.Response) (bool, error) {
				retry := false
				status := r.StatusCode()
				if status == http.StatusServiceUnavailable || status == http.StatusInternalServerError {
					retry = true
				}
				return retry, nil
			},
		)
	log.Debugf("Built basic HTTP client for route: %s", fullRestRoute)
	return client
}
