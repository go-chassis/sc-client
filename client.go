package sc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gorilla/websocket"

	"github.com/go-chassis/cari/discovery"
	"github.com/go-chassis/foundation/httpclient"
	"github.com/go-chassis/foundation/httputil"
	"github.com/go-chassis/openlog"
)

// Define constants for the client
const (
	MicroservicePath    = "/microservices"
	InstancePath        = "/instances"
	BatchInstancePath   = "/instances/action"
	SchemaPath          = "/schemas"
	HeartbeatPath       = "/heartbeat"
	ExistencePath       = "/existence"
	WatchPath           = "/watcher"
	StatusPath          = "/status"
	DependencyPath      = "/dependencies"
	PropertiesPath      = "/properties"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
	DefaultAddr         = "127.0.0.1:30100"
	AppsPath            = "/apps"
	DefaultRetryTimeout = 500 * time.Millisecond
	HeaderRevision      = "X-Resource-Revision"
	EnvProjectID        = "CSE_PROJECT_ID"
	// EnvCheckSCIInterval sc instance health check interval in second
	EnvCheckSCIInterval = "CHASSIS_SC_HEALTH_CHECK_INTERVAL"
)

// Define variables for the client
var (
	MSAPIPath     = ""
	GovernAPIPATH = ""
	TenantHeader  = "X-Domain-Name"
)
var (
	// ErrNotModified means instance is not changed
	ErrNotModified = errors.New("instance is not changed since last query")
	// ErrMicroServiceExists means service is registered
	ErrMicroServiceExists = errors.New("micro-service already exists")
	// ErrMicroServiceNotExists means service is not exists
	ErrMicroServiceNotExists = errors.New("micro-service does not exist")
	// ErrEmptyCriteria means you gave an empty list of criteria
	ErrEmptyCriteria = errors.New("batch find criteria is empty")
	ErrNil           = errors.New("input is nil")
)

// Client communicate to Service-Center
type Client struct {
	opt      Options
	client   *httpclient.Requests
	protocol string
	watchers map[string]bool
	mutex    sync.Mutex
	wsDialer *websocket.Dialer
	// record the websocket connection with the service center
	conns    map[string]*websocket.Conn
	revision string
	pool     *AddressPool
}

// URLParameter maintains the list of parameters to be added in URL
type URLParameter map[string]string

//ResetRevision reset the revision to 0
func (c *Client) ResetRevision() {
	c.revision = "0"
}

// NewClient create a the service center client
func NewClient(opt Options) (*Client, error) {
	c := &Client{
		opt:      opt,
		revision: "0",
		watchers: make(map[string]bool),
		conns:    make(map[string]*websocket.Conn),
		pool:     NewPool(),
	}
	options := &httpclient.Options{
		TLSConfig:  opt.TLSConfig,
		Compressed: opt.Compressed,
	}
	var err error
	c.client, err = httpclient.New(options)
	if err != nil {
		return nil, err
	}
	c.wsDialer = &websocket.Dialer{
		TLSClientConfig: opt.TLSConfig,
	}
	c.protocol = "https"
	if !c.opt.EnableSSL {
		c.wsDialer = websocket.DefaultDialer
		c.protocol = "http"
	}
	//Update the API Base Path based on the project
	c.updateAPIPath()
	c.pool.SetAddress(opt.Endpoints)
	return c, nil
}

func (c *Client) updateAPIPath() {
	projectID, isExist := os.LookupEnv(EnvProjectID)
	if !isExist {
		projectID = "default"
	}
	MSAPIPath = "/v4/" + projectID + "/registry"
	GovernAPIPATH = "/v4/" + projectID + "/govern"
}

// SyncEndpoints gets the endpoints of service-center in the cluster
// if your service center cluster is not behind a load balancing service like ELB,nginx etc
// then you can use this function
func (c *Client) SyncEndpoints() error {
	c.pool.Monitor()
	instances, err := c.Health()
	if err != nil {
		return fmt.Errorf("sync SC ep failed. err:%s", err.Error())
	}
	eps := make([]string, 0)
	for _, instance := range instances {
		m := getProtocolMap(instance.Endpoints)
		eps = append(eps, m["rest"])
	}
	if len(eps) != 0 {
		c.pool.SetAddress(eps)
		openlog.Info("Sync service center endpoints " + strings.Join(eps, ","))
		return nil
	}
	return fmt.Errorf("sync endpoints failed")
}

func (c *Client) formatURL(api string, querys []URLParameter, options *CallOptions) string {
	builder := URLBuilder{
		Protocol:      c.protocol,
		Host:          c.getAddress(),
		Path:          api,
		URLParameters: querys,
		CallOptions:   options,
	}
	return builder.String()
}

// GetDefaultHeaders gets the default headers for each request to be made to Service-Center
func (c *Client) GetDefaultHeaders() http.Header {
	headers := http.Header{
		HeaderContentType: []string{"application/json"},
		HeaderUserAgent:   []string{"go-client"},
		TenantHeader:      []string{"default"},
	}

	return headers
}

// httpDo makes the http request to Service-center with proper header, body and method
func (c *Client) httpDo(method string, rawURL string, headers http.Header, body []byte) (resp *http.Response, err error) {
	if len(headers) == 0 {
		headers = make(http.Header)
	}
	for k, v := range c.GetDefaultHeaders() {
		headers[k] = v
	}
	return c.client.Do(context.Background(), method, rawURL, headers, body)
}

// RegisterService registers the micro-services to Service-Center
func (c *Client) RegisterService(microService *discovery.MicroService) (string, error) {
	if microService == nil {
		return "", ErrNil
	}
	request := discovery.CreateServiceRequest{
		Service: microService,
	}

	registerURL := c.formatURL(MSAPIPath+MicroservicePath, nil, nil)
	body, err := json.Marshal(request)
	if err != nil {
		return "", NewJSONException(err, string(body))
	}

	resp, err := c.httpDo("POST", registerURL, nil, body)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("RegisterService failed, response is empty, MicroServiceName: %s", microService.ServiceName)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetExistenceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", NewJSONException(err, string(body))
		}
		microService.ServiceId = response.ServiceId
		return response.ServiceId, nil
	}
	if resp.StatusCode == 400 {
		return "", fmt.Errorf("client seems to have erred, error: %s", body)
	}
	return "", fmt.Errorf("register service failed, ServiceName/responseStatusCode/responsebody: %s/%d/%s",
		microService.ServiceName, resp.StatusCode, string(body))
}

// GetProviders gets a list of provider for a particular consumer
func (c *Client) GetProviders(consumer string, opts ...CallOption) (*MicroServiceProvideResponse, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	providersURL := c.formatURL(fmt.Sprintf("%s%s/%s/providers", MSAPIPath, MicroservicePath, consumer), nil, copts)
	resp, err := c.httpDo("GET", providersURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get Providers failed, error: %s, MicroServiceid: %s", err, consumer)
	}
	if resp == nil {
		return nil, fmt.Errorf("get Providers failed, response is empty, MicroServiceid: %s", consumer)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Get Providers failed, body is empty,  error: %s, MicroServiceid: %s", err, consumer)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p := &MicroServiceProvideResponse{}
		err = json.Unmarshal(body, p)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
	return nil, fmt.Errorf("get Providers failed, MicroServiceid: %s, response StatusCode: %d, response body: %s",
		consumer, resp.StatusCode, string(body))
}

// AddSchemas adds a schema contents to the services registered in service-center
func (c *Client) AddSchemas(microServiceID, schemaName, schemaInfo string) error {
	if microServiceID == "" {
		return errors.New("invalid micro service ID")
	}

	schemaURL := c.formatURL(fmt.Sprintf("%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID, SchemaPath), nil, nil)
	h := sha256.New()
	_, err := h.Write([]byte(schemaInfo))
	if err != nil {
		return err
	}
	request := &discovery.ModifySchemasRequest{
		Schemas: []*discovery.Schema{{
			SchemaId: schemaName,
			Schema:   schemaInfo,
			Summary:  fmt.Sprintf("%x", h.Sum(nil))}},
	}
	body, err := json.Marshal(request)
	if err != nil {
		return NewJSONException(err, string(body))
	}

	resp, err := c.httpDo("POST", schemaURL, nil, body)
	if err != nil {
		return err
	}

	if resp == nil {
		return fmt.Errorf("add schemas failed, response is empty")
	}

	if resp.StatusCode != http.StatusOK {
		return NewCommonException("add micro service schema failed. response StatusCode: %d, response body: %s",
			resp.StatusCode, string(httputil.ReadBody(resp)))
	}

	return nil
}

// GetSchema gets Schema list for the microservice from service-center
func (c *Client) GetSchema(microServiceID, schemaName string, opts ...CallOption) ([]byte, error) {
	if microServiceID == "" {
		return []byte(""), errors.New("invalid micro service ID")
	}
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	url := c.formatURL(fmt.Sprintf("%s%s/%s/%s/%s", MSAPIPath, MicroservicePath, microServiceID, "schemas", schemaName), nil, copts)
	resp, err := c.httpDo("GET", url, nil, nil)
	if err != nil {
		return []byte(""), err
	}
	if resp == nil {
		return []byte(""), fmt.Errorf("GetSchema failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}

	return []byte(""), err
}

// GetMicroServiceID gets the microserviceid by appID, serviceName and version
func (c *Client) GetMicroServiceID(appID, microServiceName, version, env string, opts ...CallOption) (string, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	url := c.formatURL(MSAPIPath+ExistencePath, []URLParameter{
		{"type": "microservice"},
		{"appId": appID},
		{"serviceName": microServiceName},
		{"version": version},
		{"env": env},
	}, copts)
	resp, err := c.httpDo("GET", url, nil, nil)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("GetMicroServiceID failed, response is empty, MicroServiceName: %s", microServiceName)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		var response discovery.GetExistenceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", NewJSONException(err, string(body))
		}
		return response.ServiceId, nil
	}
	return "", fmt.Errorf("GetMicroServiceID failed, MicroService: %s@%s#%s, response StatusCode: %d, response body: %s, URL: %s",
		microServiceName, appID, version, resp.StatusCode, string(body), url)
}

// GetAllMicroServices gets list of all the microservices registered with Service-Center
func (c *Client) GetAllMicroServices(opts ...CallOption) ([]*discovery.MicroService, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	url := c.formatURL(MSAPIPath+MicroservicePath, nil, copts)
	resp, err := c.httpDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetAllMicroServices failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetServicesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.Services, nil
	}
	return nil, fmt.Errorf("GetAllMicroServices failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// GetAllApplications returns the list of all the applications which is registered in governance-center
func (c *Client) GetAllApplications(opts ...CallOption) ([]string, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	governanceURL := c.formatURL(GovernAPIPATH+AppsPath, nil, copts)
	resp, err := c.httpDo("GET", governanceURL, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetAllApplications failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetAppsResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.AppIds, nil
	}
	return nil, fmt.Errorf("GetAllApplications failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// GetMicroService returns the microservices by ID
func (c *Client) GetMicroService(microServiceID string, opts ...CallOption) (*discovery.MicroService, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	microserviceURL := c.formatURL(fmt.Sprintf("%s%s/%s", MSAPIPath, MicroservicePath, microServiceID), nil, copts)
	resp, err := c.httpDo("GET", microserviceURL, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetMicroService failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetServiceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.Service, nil
	}
	return nil, fmt.Errorf("GetMicroService failed, MicroServiceId: %s, response StatusCode: %d, response body: %s\n, microserviceURL: %s", microServiceID, resp.StatusCode, string(body), microserviceURL)
}

//BatchFindInstances fetch instances based on service name, env, app and version
//finally it return instances grouped by service name
func (c *Client) BatchFindInstances(consumerID string, keys []*discovery.FindService, opts ...CallOption) (*discovery.BatchFindInstancesResponse, error) {
	copts := &CallOptions{Revision: c.revision}
	for _, opt := range opts {
		opt(copts)
	}
	if len(keys) == 0 {
		return nil, ErrEmptyCriteria
	}
	url := c.formatURL(MSAPIPath+BatchInstancePath, []URLParameter{
		{"type": "query"},
	}, copts)
	r := &discovery.BatchFindInstancesRequest{
		ConsumerServiceId: consumerID,
		Services:          keys,
	}
	rBody, err := json.Marshal(r)
	if err != nil {
		return nil, NewJSONException(err, string(rBody))
	}
	resp, err := c.httpDo("POST", url, http.Header{"X-ConsumerId": []string{consumerID}}, rBody)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("BatchFindInstances failed, response is empty")
	}
	body := httputil.ReadBody(resp)
	if resp.StatusCode == http.StatusOK {
		var response *discovery.BatchFindInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}

		return response, nil
	}
	return nil, fmt.Errorf("batch find failed, status %d, body %s", resp.StatusCode, body)
}

// FindMicroServiceInstances find microservice instance using consumerID, appID, name and version rule
func (c *Client) FindMicroServiceInstances(consumerID, appID, microServiceName,
	versionRule string, opts ...CallOption) ([]*discovery.MicroServiceInstance, error) {
	copts := &CallOptions{Revision: c.revision}
	for _, opt := range opts {
		opt(copts)
	}
	microserviceInstanceURL := c.formatURL(MSAPIPath+InstancePath, []URLParameter{
		{"appId": appID},
		{"serviceName": microServiceName},
		{"version": versionRule},
	}, copts)

	resp, err := c.httpDo("GET", microserviceInstanceURL, http.Header{"X-ConsumerId": []string{consumerID}}, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("FindMicroServiceInstances failed, response is empty, appID/MicroServiceName/version: %s/%s/%s", appID, microServiceName, versionRule)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		r := resp.Header.Get(HeaderRevision)
		if r != c.revision && r != "" {
			c.revision = r
			openlog.Debug("service center has new revision " + c.revision)
		}

		return response.Instances, nil
	}
	if resp.StatusCode == http.StatusNotModified {
		return nil, ErrNotModified
	}
	if resp.StatusCode == http.StatusBadRequest {
		if strings.Contains(string(body), "\"errorCode\":\"400012\"") {
			return nil, ErrMicroServiceNotExists
		}
	}
	return nil, fmt.Errorf("FindMicroServiceInstances failed, appID/MicroServiceName/version: %s/%s/%s, response StatusCode: %d, response body: %s",
		appID, microServiceName, versionRule, resp.StatusCode, string(body))
}

// RegisterMicroServiceInstance registers the microservice instance to Servive-Center
func (c *Client) RegisterMicroServiceInstance(microServiceInstance *discovery.MicroServiceInstance) (string, error) {
	if microServiceInstance == nil {
		return "", errors.New("invalid request parameter")
	}
	request := &discovery.RegisterInstanceRequest{
		Instance: microServiceInstance,
	}
	microserviceInstanceURL := c.formatURL(fmt.Sprintf("%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceInstance.ServiceId, InstancePath), nil, nil)
	body, err := json.Marshal(request)
	if err != nil {
		return "", NewJSONException(err, string(body))
	}
	resp, err := c.httpDo("POST", microserviceInstanceURL, nil, body)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("register instance failed, response is empty, MicroServiceId = %s", microServiceInstance.ServiceId)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response *discovery.RegisterInstanceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", NewJSONException(err, string(body))
		}
		return response.InstanceId, nil
	}
	return "", fmt.Errorf("register instance failed, MicroServiceId: %s, response StatusCode: %d, response body: %s",
		microServiceInstance.ServiceId, resp.StatusCode, string(body))
}

// GetMicroServiceInstances queries the service-center with provider and consumer ID and returns the microservice-instance
func (c *Client) GetMicroServiceInstances(consumerID, providerID string, opts ...CallOption) ([]*discovery.MicroServiceInstance, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s", MSAPIPath, MicroservicePath, providerID, InstancePath), nil, copts)
	resp, err := c.httpDo("GET", url, http.Header{
		"X-ConsumerId": []string{consumerID},
	}, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetMicroServiceInstances failed, response is empty, ConsumerId/ProviderId = %s%s", consumerID, providerID)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.Instances, nil
	}
	return nil, fmt.Errorf("GetMicroServiceInstances failed, ConsumerId/ProviderId: %s%s, response StatusCode: %d, response body: %s",
		consumerID, providerID, resp.StatusCode, string(body))
}

// GetAllResources retruns all the list of services, instances, providers, consumers in the service-center
func (c *Client) GetAllResources(resource string, opts ...CallOption) ([]*discovery.ServiceDetail, error) {
	copts := &CallOptions{}
	for _, opt := range opts {
		opt(copts)
	}
	url := c.formatURL(GovernAPIPATH+MicroservicePath, []URLParameter{
		{"options": resource},
	}, copts)
	resp, err := c.httpDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("GetAllResources failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetServicesInfoResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.AllServicesDetail, nil
	}
	return nil, fmt.Errorf("GetAllResources failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// Health returns the list of all the endpoints of SC with their status
func (c *Client) Health() ([]*discovery.MicroServiceInstance, error) {
	url := c.formatURL(MSAPIPath+"/health", nil, nil)
	resp, err := c.httpDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("query cluster info failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response discovery.GetInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, NewJSONException(err, string(body))
		}
		return response.Instances, nil
	}
	return nil, fmt.Errorf("query cluster info failed,  response StatusCode: %d, response body: %s",
		resp.StatusCode, string(body))
}

// Heartbeat sends the heartbeat to service-center for particular service-instance
func (c *Client) Heartbeat(microServiceID, microServiceInstanceID string) (bool, error) {
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID, HeartbeatPath), nil, nil)
	resp, err := c.httpDo("PUT", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("heartbeat failed, response is empty, MicroServiceId/MicroServiceInstanceId: %s%s", microServiceID, microServiceInstanceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// WSHeartbeat creates a web socket connection to service-center to send heartbeat.
// It relies on the ping pong mechanism of websocket to ensure the heartbeat, which is maintained by goroutines.
// After the connection is established, the communication fails and will be retried continuously. The retrial time increases exponentially.
// The callback function is used to re-register the instance.
func (c *Client) WSHeartbeat(microServiceID, microServiceInstanceID string, callback func()) error {
	err := c.setupWSConnection(microServiceID, microServiceInstanceID)
	if err != nil {
		return err
	}
	go func() {
		resetConn := func() error {
			return c.setupWSConnection(microServiceID, microServiceInstanceID)
		}
		for {
			conn := c.conns[microServiceInstanceID]
			_, _, err = conn.ReadMessage()
			if err != nil {
				openlog.Error(err.Error())
				closeErr := conn.Close()
				if closeErr != nil {
					openlog.Error(fmt.Sprintf("failed to close websocket connection %s", closeErr.Error()))
				}
				if websocket.IsCloseError(err, discovery.ErrWebsocketInstanceNotExists) {
					// If the instance does not exist, it is closed normally and should be re-registered
					callback()
				}
				// reconnection
				err = backoff.RetryNotify(
					resetConn,
					backoff.NewExponentialBackOff(),
					func(err error, duration time.Duration) {
						openlog.Error(fmt.Sprintf("failed err: %s,and it will be executed again in %v", err.Error(), duration))
					})
			}
		}
	}()
	return nil
}

// setupWSConnection create websocket connection and assign it to the map of the connection
func (c *Client) setupWSConnection(microServiceID, microServiceInstanceID string) error {
	scheme := "wss"
	if !c.opt.EnableSSL {
		scheme = "ws"
	}

	u := url.URL{
		Scheme: scheme,
		Host:   c.getAddress(),
		Path: fmt.Sprintf("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID,
			InstancePath, microServiceInstanceID, "/heartbeat"),
	}

	conn, _, err := c.wsDialer.Dial(u.String(), c.GetDefaultHeaders())
	if err != nil {
		openlog.Error(fmt.Sprintf("watching microservice dial catch an exception,microServiceID: %s, error:%s", microServiceID, err.Error()))
		return err
	}
	c.conns[microServiceInstanceID] = conn
	openlog.Info(fmt.Sprintf("%s's websocket connection established successfully", microServiceInstanceID))
	return nil
}

// UnregisterMicroServiceInstance un-registers the microservice instance from the service-center
func (c *Client) UnregisterMicroServiceInstance(microServiceID, microServiceInstanceID string) (bool, error) {
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s/%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID), nil, nil)
	resp, err := c.httpDo("DELETE", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("unregister instance failed, response is empty, MicroServiceId/MicroServiceInstanceId: %s/%s", microServiceID, microServiceInstanceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UnregisterMicroService un-registers the microservice from the service-center
func (c *Client) UnregisterMicroService(microServiceID string) (bool, error) {
	url := c.formatURL(fmt.Sprintf("%s%s/%s", MSAPIPath, MicroservicePath, microServiceID), []URLParameter{
		{"force": "1"},
	}, nil)
	resp, err := c.httpDo("DELETE", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UnregisterMicroService failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceInstanceStatus updates the microservicve instance status in service-center
func (c *Client) UpdateMicroServiceInstanceStatus(microServiceID, microServiceInstanceID, status string) (bool, error) {
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID, StatusPath), []URLParameter{
		{"value": status},
	}, nil)
	resp, err := c.httpDo("PUT", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UpdateMicroServiceInstanceStatus failed, response is empty, MicroServiceId/MicroServiceInstanceId/status: %s%s%s",
			microServiceID, microServiceInstanceID, status)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceInstanceProperties updates the microserviceinstance  prooperties in the service-center
func (c *Client) UpdateMicroServiceInstanceProperties(microServiceID, microServiceInstanceID string,
	microServiceInstance *discovery.MicroServiceInstance) (bool, error) {
	if microServiceInstance.Properties == nil {
		return false, errors.New("invalid request parameter")
	}
	request := discovery.RegisterInstanceRequest{
		Instance: microServiceInstance,
	}
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID, InstancePath, microServiceInstanceID, PropertiesPath), nil, nil)
	body, err := json.Marshal(request.Instance)
	if err != nil {
		return false, NewJSONException(err, string(body))
	}

	resp, err := c.httpDo("PUT", url, nil, body)

	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UpdateMicroServiceInstanceProperties failed, response is empty, MicroServiceId/microServiceInstanceID: %s/%s",
			microServiceID, microServiceInstanceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceProperties updates the microservice properties in the servive-center
func (c *Client) UpdateMicroServiceProperties(microServiceID string, microService *discovery.MicroService) (bool, error) {
	if microService.Properties == nil {
		return false, errors.New("invalid request parameter")
	}
	request := &discovery.CreateServiceRequest{
		Service: microService,
	}
	url := c.formatURL(fmt.Sprintf("%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID, PropertiesPath), nil, nil)
	body, err := json.Marshal(request.Service)
	if err != nil {
		return false, NewJSONException(err, string(body))
	}

	resp, err := c.httpDo("PUT", url, nil, body)

	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UpdateMicroServiceProperties failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, NewIOException(err)
		}
		return false, NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// Close closes the connection with Service-Center
func (c *Client) Close() error {
	for k, v := range c.conns {
		err := v.Close()
		if err != nil {
			return fmt.Errorf("error:%s, microServiceID = %s", err.Error(), k)
		}
		delete(c.conns, k)
	}
	return nil
}

// WatchMicroService creates a web socket connection to service-center to keep a watch on the providers for a micro-service
func (c *Client) WatchMicroService(microServiceID string, callback func(*MicroServiceInstanceChangedEvent)) error {
	if ready, ok := c.watchers[microServiceID]; !ok || !ready {
		c.mutex.Lock()
		if ready, ok := c.watchers[microServiceID]; !ok || !ready {
			c.watchers[microServiceID] = true
			scheme := "wss"
			if !c.opt.EnableSSL {
				scheme = "ws"
			}
			u := url.URL{
				Scheme: scheme,
				Host:   c.getAddress(),
				Path: fmt.Sprintf("%s%s/%s%s", MSAPIPath,
					MicroservicePath, microServiceID, WatchPath),
			}
			conn, _, err := c.wsDialer.Dial(u.String(), c.GetDefaultHeaders())
			if err != nil {
				c.watchers[microServiceID] = false
				c.mutex.Unlock()
				return fmt.Errorf("watching microservice dial catch an exception,microServiceID: %s, error:%s", microServiceID, err.Error())
			}

			c.conns[microServiceID] = conn
			go func() {
				for {
					messageType, message, err := conn.ReadMessage()
					if err != nil {
						break
					}
					if messageType == websocket.TextMessage {
						var response MicroServiceInstanceChangedEvent
						err := json.Unmarshal(message, &response)
						if err != nil {
							break
						}
						callback(&response)
					}
				}
				err = conn.Close()
				if err != nil {
					openlog.Error(err.Error())
				}
				delete(c.conns, microServiceID)
				c.startBackOff(microServiceID, callback)
			}()
		}
		c.mutex.Unlock()
	}
	return nil
}

func (c *Client) getAddress() string {
	return c.pool.GetAvailableAddress()
}

func (c *Client) startBackOff(microServiceID string, callback func(*MicroServiceInstanceChangedEvent)) {
	boff := &backoff.ExponentialBackOff{
		InitialInterval:     1000 * time.Millisecond,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         30000 * time.Millisecond,
		MaxElapsedTime:      0,
		Clock:               backoff.SystemClock,
	}
	operation := func() error {
		c.mutex.Lock()
		c.watchers[microServiceID] = false
		c.getAddress()
		c.mutex.Unlock()
		err := c.WatchMicroService(microServiceID, callback)
		if err != nil {
			return err
		}
		return nil
	}

	err := backoff.Retry(operation, boff)
	if err == nil {
		return
	}
}
