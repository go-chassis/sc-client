# Service Center client for go
This is a service center client which helps the microservice to interact with Service Center
for service-registration, discovery, instance registration etc.

This client implements all the [api's](https://rawcdn.githack.com/go-chassis/service-center/master/docs/api-docs.html) of Service Center.


# Usage
 
```go
registryClient, err := sc.NewClient(
	sc.Options{
		Addrs: []string{"127.0.0.1:30100"},
	})
```
declare and register micro service
```go
var ms = new(discovery.MicroService)
var m = make(map[string]string)

m["abc"] = "abc"
m["def"] = "def"

ms.AppId = MSList[0].AppId
ms.ServiceName = MSList[0].ServiceName
ms.Version = MSList[0].Version
ms.Environment = MSList[0].Environment
ms.Properties = m
sid, err := registryClient.RegisterService(ms)
```
declare and register instance
```go
	microServiceInstance := &discovery.MicroServiceInstance{
		Endpoints: []string{"rest://127.0.0.1:3000"},
		HostName:  hostname,
		Status:    sc.MSInstanceUP,
	}
	id, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
```