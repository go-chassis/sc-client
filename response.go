package sc

import (
	"github.com/go-chassis/cari/discovery"
)

const (
	// EventCreate is a constant of type string
	EventCreate string = "CREATE"
	// EventUpdate is a constant of type string
	EventUpdate string = "UPDATE"
	// EventDelete is a constant of type string
	EventDelete string = "DELETE"
	// EventError is a constant of type string
	EventError string = "ERROR"
	// MicorserviceUp is a constant of type string
	MicorserviceUp string = "UP"
	// MicroserviceDown is a constant of type string
	MicroserviceDown string = "DOWN"
	// MSInstanceUP is a constant of type string
	MSInstanceUP string = "UP"
	// MSIinstanceDown is a constant of type string
	MSIinstanceDown string = "DOWN"
	// CheckByHeartbeat is a constant of type string
	CheckByHeartbeat string = "push"
	// DefaultLeaseRenewalInterval is a constant of type int which declares default lease renewal time
	DefaultLeaseRenewalInterval = 30
)

// MicroServiceProvideResponse is a struct with provider information
type MicroServiceProvideResponse struct {
	Services []*discovery.MicroService `json:"providers,omitempty"`
}

// MicroServiceInstanceChangedEvent is a struct to store the Changed event information
type MicroServiceInstanceChangedEvent struct {
	Action   string                          `protobuf:"bytes,2,opt,name=action" json:"action,omitempty"`
	Key      *discovery.MicroServiceKey      `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	Instance *discovery.MicroServiceInstance `protobuf:"bytes,4,opt,name=instance" json:"instance,omitempty"`
}

type FindMicroServiceInstancesResult struct {
	Instances []*discovery.MicroServiceInstance
	Revision  string
}
