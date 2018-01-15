package model

const (
	//EventCreate is a constant of type string
	EventCreate string = "CREATE"
	EventUpdate string = "UPDATE"
	EventDelete string = "DELETE"
	EventError  string = "ERROR"

	MicorserviceUp   string = "UP"
	MicroserviceDown string = "DOWN"

	MSInstanceUP    string = "UP"
	MSIinstanceDown string = "DOWN"
	//MSI_STARTING     string = "STARTING"
	//MSI_OUTOFSERVICE string = "OUTOFSERVICE"

	CheckByHeartbeat string = "push"
	//CHECK_BY_PLATFORM             string = "pull"
	//DefaultLeaseRenewalInterval is a constant of type int which declares default lease renewal time
	DefaultLeaseRenewalInterval = 30
)

// MicroServiceKey is a struct with key information about Microservice
type MicroServiceKey struct {
	Tenant      string `protobuf:"bytes,1,opt,name=tenant" json:"tenant,omitempty"`
	Project     string `protobuf:"bytes,2,opt,name=project" json:"project,omitempty"`
	AppID       string `protobuf:"bytes,3,opt,name=appId" json:"appId,omitempty"`
	ServiceName string `protobuf:"bytes,4,opt,name=serviceName" json:"serviceName,omitempty"`
	Version     string `protobuf:"bytes,5,opt,name=version" json:"version,omitempty"`
	Stage       string `protobuf:"bytes,6,opt,name=stage" json:"stage,omitempty"`
	ins         []MicroServiceInstance
}

// ServicePath is a struct with path and property information
type ServicePath struct {
	Path     string            `protobuf:"bytes,1,opt,name=path" json:"path,omitempty"`
	Property map[string]string `protobuf:"bytes,2,rep,name=property" json:"property,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// MicroService is a struct with all detailed information of microservies
type MicroService struct {
	ServiceID   string   `protobuf:"bytes,1,opt,name=serviceId" json:"serviceId,omitempty"`
	AppID       string   `protobuf:"bytes,2,opt,name=appId" json:"appId,omitempty"`
	ServiceName string   `protobuf:"bytes,3,opt,name=serviceName" json:"serviceName,omitempty"`
	Version     string   `protobuf:"bytes,4,opt,name=version" json:"version,omitempty"`
	Description string   `protobuf:"bytes,5,opt,name=description" json:"description,omitempty"`
	Level       string   `protobuf:"bytes,6,opt,name=level" json:"level,omitempty"`
	Schemas     []string `protobuf:"bytes,7,rep,name=schemas" json:"schemas,omitempty"`
	Paths       []*ServicePath            `protobuf:"bytes,10,rep,name=paths" json:"paths,omitempty"`
	Status      string                    `protobuf:"bytes,8,opt,name=status" json:"status,omitempty"`
	Properties  map[string]string         `protobuf:"bytes,9,rep,name=properties" json:"properties,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Timestamp   string                    `protobuf:"bytes,11,opt,name=timestamp" json:"timestamp,omitempty"`
	Providers   []*DependencyMicroService `protobuf:"bytes,12,rep,name=providers" json:"providers,omitempty"`
	Framework   *Framework                `protobuf:"bytes,13,opt,name=framework" json:"framework,omitempty"`
	RegisterBy  string                    `protobuf:"bytes,14,opt,name=registerBy" json:"registerBy,omitempty"`
}

// Framework is a struct which contains name and version of the Framework
type Framework struct {
	Name    string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Version string `protobuf:"bytes,1,opt,name=version" json:"version,omitempty"`
}

// HealthCheck is struct with contains mode, port and interval of sc from which it needs to poll information
type HealthCheck struct {
	Mode     string `protobuf:"bytes,1,opt,name=mode" json:"mode,omitempty"`
	Port     int32  `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
	Interval int32  `protobuf:"varint,3,opt,name=interval" json:"interval,omitempty"`
	Times    int32  `protobuf:"varint,4,opt,name=times" json:"times,omitempty"`
}

// DataCenterInfo is a struct with containes the zone information of the data center
type DataCenterInfo struct {
	Name          string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
	Region        string `protobuf:"bytes,2,opt,name=region" json:"region,omitempty"`
	AvailableZone string `protobuf:"bytes,3,opt,name=availableZone" json:"availableZone,omitempty"`
}

// MicroServiceInstance is a struct to store all the detailed information about micro-service information
type MicroServiceInstance struct {
	InstanceID string            `protobuf:"bytes,1,opt,name=instanceId" json:"instanceId,omitempty"`
	ServiceID  string            `protobuf:"bytes,2,opt,name=serviceId" json:"serviceId,omitempty"`
	Endpoints  []string          `protobuf:"bytes,3,rep,name=endpoints" json:"endpoints,omitempty"`
	HostName   string            `protobuf:"bytes,4,opt,name=hostName" json:"hostName,omitempty"`
	Status     string            `protobuf:"bytes,5,opt,name=status" json:"status,omitempty"`
	Properties map[string]string `protobuf:"bytes,6,rep,name=properties" json:"properties,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	HealthCheck    *HealthCheck      `protobuf:"bytes,7,opt,name=healthCheck" json:"healthCheck,omitempty"`
	Timestamp      string            `protobuf:"bytes,8,opt,name=timestamp" json:"timestamp,omitempty"`
	DataCenterInfo *DataCenterInfo   `protobuf:"bytes,9,opt,name=dataCenterInfo" json:"dataCenterInfo,omitempty"`
	Environment    string            `protobuf:"bytes,10,opt,name=environment" json:"environment,omitempty"`
}

// MicroServiceInstanceChangedEvent is a struct to store the Changed event information
type MicroServiceInstanceChangedEvent struct {
	Action   string                `protobuf:"bytes,2,opt,name=action" json:"action,omitempty"`
	Key      *MicroServiceKey      `protobuf:"bytes,3,opt,name=key" json:"key,omitempty"`
	Instance *MicroServiceInstance `protobuf:"bytes,4,opt,name=instance" json:"instance,omitempty"`
}

// MicroServiceInstanceKey is a struct to key ID's of the microservice
type MicroServiceInstanceKey struct {
	InstanceID string `protobuf:"bytes,1,opt,name=instanceId" json:"instanceId,omitempty"`
	ServiceID  string `protobuf:"bytes,2,opt,name=serviceId" json:"serviceId,omitempty"`
}

// DependencyMicroService is a struct to keep dependency information for the microservice
type DependencyMicroService struct {
	AppID       string `protobuf:"bytes,1,opt,name=appId" json:"appId,omitempty"`
	ServiceName string `protobuf:"bytes,2,opt,name=serviceName" json:"serviceName,omitempty"`
	Version     string `protobuf:"bytes,3,opt,name=version" json:"version,omitempty"`
}

// MicroServiceDependency is a struct to keep the all the dependency information
type MicroServiceDependency struct {
	Consumer  *DependencyMicroService   `protobuf:"bytes,1,opt,name=consumer" json:"consumer,omitempty"`
	Providers []*DependencyMicroService `protobuf:"bytes,2,rep,name=providers" json:"providers,omitempty"`
}

// GetServicesInfoResponse is a struct to keep all the list of services.
type GetServicesInfoResponse struct {
	AllServicesDetail []*ServiceDetail `protobuf:"bytes,2,rep,name=allServicesDetail" json:"allServicesDetail,omitempty"`
}

// ServiceDetail is a struct to store all the relevant information for a microservice
type ServiceDetail struct {
	MicroService         *MicroService           `protobuf:"bytes,1,opt,name=microSerivce" json:"microSerivce,omitempty"`
	Instances            []*MicroServiceInstance `protobuf:"bytes,2,rep,name=instances" json:"instances,omitempty"`
	Providers            []*MicroService         `protobuf:"bytes,5,rep,name=providers" json:"providers,omitempty"`
	Consumers            []*MicroService         `protobuf:"bytes,6,rep,name=consumers" json:"consumers,omitempty"`
	Tags                 map[string]string       `protobuf:"bytes,7,rep,name=tags" json:"tags,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	MicroServiceVersions []string                `protobuf:"bytes,8,rep,name=microServiceVersions" json:"microServiceVersions,omitempty"`
}
