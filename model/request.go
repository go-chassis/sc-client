package model

// MicroServiceRequest is a struct with microservice information
type MicroServiceRequest struct {
	Service *MicroService `json:"service"`
}

// MicroServiceInstanceRequest is struct with microservice instance information
type MicroServiceInstanceRequest struct {
	Instance *MicroServiceInstance `json:"instance"`
}

// MicroServiceInstanceSchemaUpdateRequest is a struct with Schema Content
type MicroServiceInstanceSchemaUpdateRequest struct {
	SchemaContent string `json:"schema"`
}

// MircroServiceDependencyRequest is a struct with dependencies request
type MircroServiceDependencyRequest struct {
	Dependencies []*MicroServiceDependency `json:"dependencies"`
}
