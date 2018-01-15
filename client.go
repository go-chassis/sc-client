package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ServiceComb/go-chassis/core/common"
	"github.com/ServiceComb/go-chassis/core/lager"
	"github.com/ServiceComb/go-sc-client/model"
	"github.com/ServiceComb/http-client"
	"github.com/cenkalti/backoff"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Define constants for the client
const (
	MicroservicePath    = "/microservices"
	InstancePath        = "/instances"
	SchemaPath          = "/schemas"
	HeartbeatPath       = "/heartbeat"
	ExistencePath       = "/existence"
	WatchPath           = "/watcher"
	StatusPath          = "/status"
	DependencyPath      = "/dependencies"
	PropertiesPath      = "/properties"
	HeaderContentType   = "Content-Type"
	HeaderUserAgent     = "User-Agent"
	protocolSymbol      = "://"
	DefaultAddr         = "127.0.0.1:30100"
	AppsPath            = "/apps"
	DefaultRetryTimeout = 500 * time.Millisecond
)

// Define variables for the client
var (
	MSAPIPath     = ""
	TenantHeader  = ""
	GovernAPIPATH = ""
)

// RegistryClient is a structure for the client to communicate to Service-Center
type RegistryClient struct {
	Config     *RegistryConfig
	client     *httpclient.URLClient
	protocol   string
	watchers   map[string]bool
	mutex      sync.Mutex
	wsDialer   *websocket.Dialer
	conns      map[string]*websocket.Conn
	apiVersion string
}

// RegistryConfig is a structure to store registry configurations like address of cc, ssl configurations and tenant name
type RegistryConfig struct {
	Addresses []string
	SSL       bool
	Tenant    string
}

// URLParameter maintains the list of parameters to be added in URL
type URLParameter map[string]string

// Initialize initializes the Registry Client
func (c *RegistryClient) Initialize(opt Options) (err error) {
	c.Config = &RegistryConfig{
		Addresses: opt.Addrs,
		SSL:       opt.EnableSSL,
		Tenant:    opt.ConfigTenant,
	}

	options := &httpclient.URLClientOption{
		SSLEnabled: opt.EnableSSL,
		TLSConfig:  opt.TLSConfig,
		Compressed: opt.Compressed,
		Verbose:    opt.Verbose,
	}
	c.watchers = make(map[string]bool)
	c.conns = make(map[string]*websocket.Conn)
	c.protocol = "https"
	c.wsDialer = &websocket.Dialer{
		TLSClientConfig: opt.TLSConfig,
	}
	if !c.Config.SSL {
		c.wsDialer = websocket.DefaultDialer
		c.protocol = "http"
	}
	c.client, err = httpclient.GetURLClient(options)
	if err != nil {
		return err
	}

	//Set the API Version based on the value set in chassis.yaml cse.service.registry.api.version
	//Default Value Set to V4
	switch opt.Version {
	case "v3":
		c.apiVersion = "v3"
	case "V3":
		c.apiVersion = "v3"
	case "v4":
		c.apiVersion = "v4"
	case "V4":
		c.apiVersion = "v4"
	default:
		c.apiVersion = "v4"
	}
	//Update the API Base Path based on the Version
	c.updateAPIPath()

	return nil
}

// updateAPIPath Updates the Base PATH anf HEADERS Based on the version of SC used.
func (c *RegistryClient) updateAPIPath() {

	//Check for the env Name in Container to get Domain Name
	//Default value is  "default"
	projectID, isExsist := os.LookupEnv(common.EnvProjectID)
	if !isExsist {
		projectID = "default"
	}
	switch c.apiVersion {
	case "v4":
		MSAPIPath = "/v4/" + projectID + "/registry"
		TenantHeader = "X-Domain-Name"
		GovernAPIPATH = "/v4/" + projectID + "/govern"
		lager.Logger.Info("Use Service center v4")
	case "v3":
		MSAPIPath = "/registry/v3"
		TenantHeader = "X-Tenant-Name"
		GovernAPIPATH = "/registry/v3"
		lager.Logger.Info("Use Service center v3")
	default:
		MSAPIPath = "/v4/" + projectID + "/registry"
		TenantHeader = "X-Domain-Name"
		GovernAPIPATH = "/v4/" + projectID + "/govern"
		lager.Logger.Info("Use Service center v4")
	}
}

// SyncEndpoints gets the endpoints of service-center in the cluster
func (c *RegistryClient) SyncEndpoints() error {
	instances, err := c.Health()
	if err != nil {
		return fmt.Errorf("sync SC ep failed.err:%s", err.Error())
	}
	eps := []string{}
	for _, instance := range instances {
		m := getProtocolMap(instance.Endpoints)
		eps = append(eps, m["rest"])
	}
	if len(eps) != 0 {
		c.Config.Addresses = eps
		lager.Logger.Info("Sync service center endpoints " + strings.Join(eps, ","))
		return nil
	}
	return fmt.Errorf("Sync endpoints failed")
}

func (c *RegistryClient) formatURL(format string, v ...interface{}) string {
	return fmt.Sprintf("%s://%s%s", c.protocol, c.getAddress(), fmt.Sprintf(format, v...))

}

func (c *RegistryClient) encodeParams(params []URLParameter) string {
	encoded := []string{}
	for _, param := range params {
		for k, v := range param {
			encoded = append(encoded, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
		}
	}
	return strings.Join(encoded, "&")
}

// GetDefaultHeaders gets the default headers for each request to be made to Service-Center
func (c *RegistryClient) GetDefaultHeaders() http.Header {
	headers := http.Header{
		HeaderContentType: []string{"application/json"},
		HeaderUserAgent:   []string{"cse-serviceregistry-client/1.0.0"},
		TenantHeader:      []string{"default"},
	}
	return headers
}

// HTTPDo makes the http request to Service-center with proper header, body and method
func (c *RegistryClient) HTTPDo(method string, rawURL string, headers http.Header, body []byte) (resp *http.Response, err error) {
	if len(headers) == 0 {
		headers = make(http.Header)
	}
	for k, v := range c.GetDefaultHeaders() {
		headers[k] = v
	}
	return c.client.HttpDo(method, rawURL, headers, body)
}

// RegisterService registers the micro-services to Service-Center
func (c *RegistryClient) RegisterService(microService *model.MicroService) (string, error) {
	if microService == nil {
		return "", errors.New("invalid request MicroService parameter")
	}
	if microService.Version == "" {
		microService.Version = "0.1"
	}
	request := &model.MicroServiceRequest{
		Service: microService,
	}

	registerURL := c.formatURL("%s%s", MSAPIPath, MicroservicePath)
	body, err := json.Marshal(request)
	if err != nil {
		return "", model.NewJSONException(err, string(body))
	}

	resp, err := c.HTTPDo("POST", registerURL, nil, body)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("RegisterService failed, response is empty, MicroServiceName: %s", microService.ServiceName)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.ExistenceIDResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", model.NewJSONException(err, string(body))
		}
		microService.ServiceID = response.ServiceID
		return response.ServiceID, nil
	}
	return "", fmt.Errorf("RegisterService failed, MicroServiceName/responseStatusCode/responsebody: %s/%d/%s",
		microService.ServiceName, resp.StatusCode, string(body))
}

// GetProviders gets a list of provider for a particular consumer
func (c *RegistryClient) GetProviders(consumer string) (*model.MicroServiceProvideresponse, error) {
	providersURL := c.formatURL("%s%s/%s/providers", MSAPIPath, MicroservicePath, consumer)
	resp, err := c.HTTPDo("GET", providersURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Get Providers failed, error: %s, MicroServiceid: %s", err, consumer)
	}
	if resp == nil {
		return nil, fmt.Errorf("Get Providers failed, response is empty, MicroServiceid: %s", consumer)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Get Providers failed, body is empty,  error: %s, MicroServiceid: %s", err, consumer)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		p := &model.MicroServiceProvideresponse{}
		err = json.Unmarshal(body, p)
		if err != nil {
			return nil, err
		}
		return p, nil
	}
	return nil, fmt.Errorf("Get Providers failed, MicroServiceid: %s, response StatusCode: %d, response body: %s",
		consumer, resp.StatusCode, string(body))
}

// AddDependencies ： 注册微服务的依赖关系
func (c *RegistryClient) AddDependencies(request *model.MircroServiceDependencyRequest) error {
	if request == nil {
		return errors.New("invalid request parameter")
	}
	dependenciesURL := c.formatURL("%s%s", MSAPIPath, DependencyPath)

	body, err := json.Marshal(request)
	if err != nil {
		return model.NewJSONException(err, string(body))
	}

	resp, err := c.HTTPDo("PUT", dependenciesURL, nil, body)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("AddDependencies failed, response is empty")
	}
	if resp.StatusCode != http.StatusOK {
		return model.NewCommonException("add microservice dependencies failed. response StatusCode: %d, response body: %s",
			resp.StatusCode, string(body))
	}
	return nil
}

// AddSchemas adds a schema contents to the services registered in service-center
func (c *RegistryClient) AddSchemas(microServiceID, schemaName, schemaInfo string) error {
	if microServiceID == "" {
		return errors.New("invalid microserviceID")
	}

	schemaURL := c.formatURL("%s%s/%s%s/%s", MSAPIPath, MicroservicePath, microServiceID, SchemaPath, schemaName)
	request := &model.MicroServiceInstanceSchemaUpdateRequest{
		SchemaContent: schemaInfo,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return model.NewJSONException(err, string(body))
	}

	resp, err := c.HTTPDo("PUT", schemaURL, nil, body)
	if err != nil {
		return err
	}

	if resp == nil {
		return fmt.Errorf("Addschemas failed, response is empty")
	}

	if resp.StatusCode != http.StatusOK {
		return model.NewCommonException("add microservice schema failed. response StatusCode: %d, response body: %s",
			resp.StatusCode, string(body))
	}

	return nil
}

// GetSchema gets Schema list for the microservice from service-center
func (c *RegistryClient) GetSchema(microServiceID, schemaName string) ([]byte, error) {
	if microServiceID == "" {
		return []byte(""), errors.New("invalid microserviceID")
	}
	url := c.formatURL("%s%s/%s/%s/%s", MSAPIPath, MicroservicePath, microServiceID, "schemas", schemaName)
	resp, err := c.HTTPDo("GET", url, nil, nil)
	if err != nil {
		return []byte(""), err
	}
	if resp == nil {
		return []byte(""), fmt.Errorf("GetSchema failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte(""), model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}

	return []byte(""), err
}

// GetMicroServiceID gets the microserviceid by appID, serviceName and version
func (c *RegistryClient) GetMicroServiceID(appID, microServiceName, version string) (string, error) {
	url := c.formatURL("%s%s?%s", MSAPIPath, ExistencePath, c.encodeParams([]URLParameter{
		{"type": "microservice"},
		{"appId": appID},
		{"serviceName": microServiceName},
		{"version": version},
	}))
	resp, err := c.HTTPDo("GET", url, nil, nil)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("GetMicroServiceID failed, response is empty, MicroServiceName: %s", microServiceName)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		var response model.ExistenceIDResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", model.NewJSONException(err, string(body))
		}
		return response.ServiceID, nil
	}
	return "", fmt.Errorf("GetMicroServiceID failed, MicroService: %s@%s#%s, response StatusCode: %d, response body: %s, URL: %s",
		microServiceName, appID, version, resp.StatusCode, string(body), url)
}

// GetAllMicroServices gets list of all the microservices registered with Service-Center
func (c *RegistryClient) GetAllMicroServices() ([]*model.MicroService, error) {
	url := c.formatURL("%s%s", MSAPIPath, MicroservicePath)
	resp, err := c.HTTPDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetAllMicroServices failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.MicroServciesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.Services, nil
	}
	return nil, fmt.Errorf("GetAllMicroServices failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// GetAllApplications returns the list of all the applications which is registered in governance-center
func (c *RegistryClient) GetAllApplications() ([]string, error) {
	governanceURL := c.formatURL("%s%s", GovernAPIPATH, AppsPath)
	resp, err := c.HTTPDo("GET", governanceURL, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetAllApplications failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.AppsResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.AppIds, nil
	}
	return nil, fmt.Errorf("GetAllApplications failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// GetMicroService returns the microservices by ID
func (c *RegistryClient) GetMicroService(microServiceID string) (*model.MicroService, error) {
	microserviceURL := c.formatURL("%s%s/%s", MSAPIPath, MicroservicePath, microServiceID)
	resp, err := c.HTTPDo("GET", microserviceURL, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("GetMicroService failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.MicroServiceResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.Service, nil
	}
	return nil, fmt.Errorf("GetMicroService failed, MicroServiceId: %s, response StatusCode: %d, response body: %s\n, microserviceURL: %s", microServiceID, resp.StatusCode, string(body), microserviceURL)
}

// FindMicroServiceInstances find microservice instance using consumerID, appID, name and version rule
func (c *RegistryClient) FindMicroServiceInstances(consumerID, appID, microServiceName, versionRule, stage string) ([]*model.MicroServiceInstance, error) {
	microserviceInstanceURL := c.formatURL("%s%s?%s", MSAPIPath, InstancePath, c.encodeParams([]URLParameter{
		{"appId": appID},
		{"serviceName": microServiceName},
		{"version": versionRule},
		{"env": stage},
	}))
	resp, err := c.HTTPDo("GET", microserviceInstanceURL, http.Header{"X-ConsumerId": []string{consumerID}}, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("FindMicroServiceInstances failed, response is empty, appID/MicroServiceName/version/stage: %s/%s/%s/%s", appID, microServiceName, versionRule, stage)
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.MicroServiceInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.Instances, nil
	}
	return nil, fmt.Errorf("FindMicroServiceInstances failed, appID/MicroServiceName/version/stage: %s/%s/%s/%s, response StatusCode: %d, response body: %s",
		appID, microServiceName, versionRule, stage, resp.StatusCode, string(body))
}

// RegisterMicroServiceInstance registers the microservice instance to Servive-Center
func (c *RegistryClient) RegisterMicroServiceInstance(microServiceInstance *model.MicroServiceInstance) (string, error) {
	if microServiceInstance == nil {
		return "", errors.New("invalid request parameter")
	}
	request := &model.MicroServiceInstanceRequest{
		Instance: microServiceInstance,
	}
	microserviceInstanceURL := c.formatURL("%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceInstance.ServiceID, InstancePath)
	body, err := json.Marshal(request)
	if err != nil {
		return "", model.NewJSONException(err, string(body))
	}
	resp, err := c.HTTPDo("POST", microserviceInstanceURL, nil, body)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", fmt.Errorf("RegisterMicroServiceInstance failed, response is empty, MicroServiceId = %s", microServiceInstance.ServiceID)
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.ExistenceIDResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", model.NewJSONException(err, string(body))
		}
		return response.InstanceID, nil
	}
	return "", fmt.Errorf("RegisterMicroServiceInstance failed, MicroServiceId: %s, response StatusCode: %d, response body: %s",
		microServiceInstance.ServiceID, resp.StatusCode, string(body))
}

// GetMicroServiceInstances queries the service-center with provider and consumer ID and returns the microservice-instance
func (c *RegistryClient) GetMicroServiceInstances(consumerID, providerID string) ([]*model.MicroServiceInstance, error) {
	url := c.formatURL("%s%s/%s%s", MSAPIPath, MicroservicePath, providerID, InstancePath)
	resp, err := c.HTTPDo("GET", url, http.Header{
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
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.MicroServiceInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.Instances, nil
	}
	return nil, fmt.Errorf("GetMicroServiceInstances failed, ConsumerId/ProviderId: %s%s, response StatusCode: %d, response body: %s",
		consumerID, providerID, resp.StatusCode, string(body))
}

// GetAllResources retruns all the list of services, instances, providers, consumers in the service-center
func (c *RegistryClient) GetAllResources(resource string) ([]*model.ServiceDetail, error) {
	url := c.formatURL("%s/%s?options=%s", GovernAPIPATH, "microservices", resource)
	resp, err := c.HTTPDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("GetAllResources failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.GetServicesInfoResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.AllServicesDetail, nil
	}
	return nil, fmt.Errorf("GetAllResources failed, response StatusCode: %d, response body: %s", resp.StatusCode, string(body))
}

// Health returns the list of all the endpoints of SC with their status
func (c *RegistryClient) Health() ([]*model.MicroServiceInstance, error) {
	url := ""
	if c.apiVersion == "v4" {
		url = c.formatURL("/%s/%s", MSAPIPath, "health")
	} else {
		url = c.formatURL("/%s", "health")
	}

	resp, err := c.HTTPDo("GET", url, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("query cluster info failed, response is empty")
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, model.NewIOException(err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var response model.MicroServiceInstancesResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			return nil, model.NewJSONException(err, string(body))
		}
		return response.Instances, nil
	}
	return nil, fmt.Errorf("query cluster info failed,  response StatusCode: %d, response body: %s",
		resp.StatusCode, string(body))
}

// Heartbeat sends the heartbeat to service-senter for particular service-instance
func (c *RegistryClient) Heartbeat(microServiceID, microServiceInstanceID string) (bool, error) {
	url := c.formatURL("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID, HeartbeatPath)
	resp, err := c.HTTPDo("PUT", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("Heartbeat failed, response is empty, MicroServiceId/MicroServiceInstanceId: %s%s", microServiceID, microServiceInstanceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UnregisterMicroServiceInstance un-registers the microservice instance from the service-center
func (c *RegistryClient) UnregisterMicroServiceInstance(microServiceID, microServiceInstanceID string) (bool, error) {
	url := c.formatURL("%s%s/%s%s/%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID)
	resp, err := c.HTTPDo("DELETE", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UnregisterMicroServiceInstance failed, response is empty, MicroServiceId/MicroServiceInstanceId: %s/%s", microServiceID, microServiceInstanceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UnregisterMicroService un-registers the microservice from the service-center
func (c *RegistryClient) UnregisterMicroService(microServiceID string) (bool, error) {
	url := c.formatURL("%s%s/%s?force=1", MSAPIPath, MicroservicePath, microServiceID)
	resp, err := c.HTTPDo("DELETE", url, nil, nil)
	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UnregisterMicroService failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceInstanceStatus updates the microservicve instance status in service-center
func (c *RegistryClient) UpdateMicroServiceInstanceStatus(microServiceID, microServiceInstanceID, status string) (bool, error) {
	url := c.formatURL("%s%s/%s%s/%s%s?value=%s", MSAPIPath, MicroservicePath, microServiceID,
		InstancePath, microServiceInstanceID, StatusPath, status)
	resp, err := c.HTTPDo("PUT", url, nil, nil)
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
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceInstanceProperties updates the microserviceinstance  prooperties in the service-center
func (c *RegistryClient) UpdateMicroServiceInstanceProperties(microServiceID, microServiceInstanceID string, microServiceInstance *model.MicroServiceInstance) (bool, error) {
	if microServiceInstance.Properties == nil {
		return false, errors.New("invalid request parameter")
	}
	request := &model.MicroServiceInstanceRequest{
		Instance: microServiceInstance,
	}
	url := c.formatURL("%s%s/%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID, InstancePath, microServiceInstanceID, PropertiesPath)
	body, err := json.Marshal(request.Instance)
	if err != nil {
		return false, model.NewJSONException(err, string(body))
	}

	resp, err := c.HTTPDo("PUT", url, nil, body)

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
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// UpdateMicroServiceProperties updates the microservice properties in the servive-center
func (c *RegistryClient) UpdateMicroServiceProperties(microServiceID string, microService *model.MicroService) (bool, error) {
	if microService.Properties == nil {
		return false, errors.New("invalid request parameter")
	}
	request := &model.MicroServiceRequest{
		Service: microService,
	}
	url := c.formatURL("%s%s/%s%s", MSAPIPath, MicroservicePath, microServiceID, PropertiesPath)
	body, err := json.Marshal(request.Service)
	if err != nil {
		return false, model.NewJSONException(err, string(body))
	}

	resp, err := c.HTTPDo("PUT", url, nil, body)

	if err != nil {
		return false, err
	}
	if resp == nil {
		return false, fmt.Errorf("UpdateMicroServiceProperties failed, response is empty, MicroServiceId: %s", microServiceID)
	}
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, model.NewIOException(err)
		}
		return false, model.NewCommonException("result: %d %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// Close closes the connection with Service-Center
func (c *RegistryClient) Close() error {
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
func (c *RegistryClient) WatchMicroService(microServiceID string, callback func(*model.MicroServiceInstanceChangedEvent)) error {
	if ready, ok := c.watchers[microServiceID]; !ok || !ready {
		c.mutex.Lock()
		if ready, ok := c.watchers[microServiceID]; !ok || !ready {
			c.watchers[microServiceID] = true
			scheme := "wss"
			if !c.Config.SSL {
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
				c.mutex.Unlock()
				return fmt.Errorf("watching microservice dial catch an exception,microServiceID: %s, error:%s", microServiceID, err.Error())
			}

			c.conns[microServiceID] = conn
			go func() error {
				for {
					messageType, message, err := conn.ReadMessage()
					if err != nil {
						break
					}
					if messageType == websocket.TextMessage {
						var response model.MicroServiceInstanceChangedEvent
						err := json.Unmarshal(message, &response)
						if err != nil {
							break
						}
						callback(&response)
					}
				}
				err = conn.Close()
				if err != nil {
					return fmt.Errorf("Conn close failed,microServiceID: %s, error:%s", microServiceID, err.Error())
				}
				delete(c.conns, microServiceID)
				c.startBackOff(microServiceID, callback)
				return nil
			}()
		}
		c.mutex.Unlock()
	}
	return nil
}

func (c *RegistryClient) getAddress() string {
	next := RoundRobin(c.Config.Addresses)
	addr, err := next()
	if err != nil {
		return DefaultAddr
	}
	return addr
}

func (c *RegistryClient) startBackOff(microServiceID string, callback func(*model.MicroServiceInstanceChangedEvent)) {
	boff := getBackOff("Exponential")
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

func getBackOff(backoffType string) backoff.BackOff {
	switch backoffType {
	case "Exponential":
		return &backoff.ExponentialBackOff{
			InitialInterval:     1000 * time.Millisecond,
			RandomizationFactor: backoff.DefaultRandomizationFactor,
			Multiplier:          backoff.DefaultMultiplier,
			MaxInterval:         30000 * time.Millisecond,
			MaxElapsedTime:      10000 * time.Millisecond,
			Clock:               backoff.SystemClock,
		}
	case "Constant":
		return backoff.NewConstantBackOff(DefaultRetryTimeout * time.Millisecond)
	case "Zero":
		return &backoff.ZeroBackOff{}
	default:
		return backoff.NewConstantBackOff(DefaultRetryTimeout * time.Millisecond)
	}
}

func getProtocolMap(eps []string) map[string]string {
	m := make(map[string]string)
	for _, ep := range eps {
		temp := strings.Split(ep, protocolSymbol)
		m[temp[0]] = temp[1]
	}
	return m
}
