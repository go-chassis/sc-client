package model

// ExistenceIDResponse is a structure for microservice with serviceID, schemaID and InstanceID
type ExistenceIDResponse struct {
	ServiceID  string `json:"serviceId,omitempty"`
	SchemaID   string `json:"schemaId,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
}

// MicroServiceResponse is a struct with service information
type MicroServiceResponse struct {
	Service *MicroService `json:"service,omitempty"`
}

// MicroServciesResponse is a struct with services information
type MicroServciesResponse struct {
	Services []*MicroService `json:"services,omitempty"`
}

// MicroServiceInstancesResponse is a struct with instances information
type MicroServiceInstancesResponse struct {
	Instances []*MicroServiceInstance `json:"instances,omitempty"`
}

// MicroServiceProvideresponse is a struct with provider information
type MicroServiceProvideresponse struct {
	Services []*MicroService `json:"providers,omitempty"`
}

// AppsResponse is a struct with list of app ID's
type AppsResponse struct {
	AppIds []string `json:"appIds,omitempty"`
}
