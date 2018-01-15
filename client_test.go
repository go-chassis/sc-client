package client_test

import (
	"github.com/ServiceComb/go-sc-client"
	"github.com/ServiceComb/go-sc-client/model"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/ServiceComb/go-chassis/core/lager"
	"os"
)

func TestLoadbalance(t *testing.T) {
	t.Log("Testing Round robin function")
	var sArr []string

	sArr = append(sArr, "s1")
	sArr = append(sArr, "s2")

	next := client.RoundRobin(sArr)
	_, err := next()
	assert.NoError(t, err)
}

func TestLoadbalanceEmpty(t *testing.T) {
	t.Log("Testing Round robin with empty endpoint arrays")
	var sArrEmpty []string

	next := client.RoundRobin(sArrEmpty)
	_, err := next()
	assert.Error(t, err)

}

func TestClientInitializeHttpErr(t *testing.T) {
	t.Log("Testing for HTTPDo function with errors")

	lager.Initialize("", "INFO", "", "size", true, 1, 10, 7)

	hostname, err := os.Hostname()
	if err != nil {
		lager.Logger.Error("Get hostname failed.", err)
		return
	}

	registryClient := &client.RegistryClient{}

	microServiceInstance := &model.MicroServiceInstance{
		Endpoints:   []string{"rest://127.0.0.1:3000"},
		HostName:    hostname,
		Status:      model.MSInstanceUP,
		Environment: "production",
	}

	err = registryClient.Initialize(
		client.Options{
			Addrs: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	err = registryClient.SyncEndpoints()
	assert.NoError(t, err)

	httpHeader := registryClient.GetDefaultHeaders()
	assert.NotEmpty(t, httpHeader)

	resp, err := registryClient.HTTPDo("GET", "fakeRawUrl", httpHeader, []byte("fakeBody"))
	assert.Empty(t, resp)
	assert.Error(t, err)

	MSList, err := registryClient.GetAllMicroServices()
	assert.NotEmpty(t, MSList)
	assert.NoError(t, err)

	f1 := func(*model.MicroServiceInstanceChangedEvent) {}
	err = registryClient.WatchMicroService(MSList[0].ServiceID, f1)
	assert.NoError(t, err)

	var ms *model.MicroService = new(model.MicroService)
	var msdepreq *model.MircroServiceDependencyRequest = new(model.MircroServiceDependencyRequest)
	var msdepArr []*model.MicroServiceDependency
	var msdep1 *model.MicroServiceDependency = new(model.MicroServiceDependency)
	var msdep2 *model.MicroServiceDependency = new(model.MicroServiceDependency)
	var dep *model.DependencyMicroService = new(model.DependencyMicroService)
	var m map[string]string = make(map[string]string)

	m["abc"] = "abc"
	m["def"] = "def"

	dep.AppID = "appid"

	msdep1.Consumer = dep
	msdep2.Consumer = dep

	msdepArr = append(msdepArr, msdep1)
	msdepArr = append(msdepArr, msdep2)

	ms.AppID = "1"
	ms.ServiceID = MSList[0].ServiceID
	ms.ServiceName = MSList[0].ServiceName
	ms.Properties = m

	msdepreq.Dependencies = msdepArr
	s1, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
	assert.Empty(t, s1)
	assert.Error(t, err)

	s1, err = registryClient.RegisterMicroServiceInstance(nil)
	assert.Empty(t, s1)
	assert.Error(t, err)

	msArr, err := registryClient.GetMicroServiceInstances("fakeConsumerID", "fakeProviderID")
	assert.Empty(t, msArr)
	assert.Error(t, err)

	msArr, err = registryClient.Health()
	assert.NotEmpty(t, msArr)
	assert.NoError(t, err)

	b, err := registryClient.UpdateMicroServiceProperties(MSList[0].ServiceID, ms)
	assert.Equal(t, true, b)
	assert.NoError(t, err)

	f1 = func(*model.MicroServiceInstanceChangedEvent) {}
	err = registryClient.WatchMicroService(MSList[0].ServiceID, f1)
	assert.NoError(t, err)

	f1 = func(*model.MicroServiceInstanceChangedEvent) {}
	err = registryClient.WatchMicroService("", f1)
	assert.Error(t, err)

	f1 = func(*model.MicroServiceInstanceChangedEvent) {}
	err = registryClient.WatchMicroService(MSList[0].ServiceID, nil)
	assert.NoError(t, err)

	str, err := registryClient.RegisterService(ms)
	assert.Empty(t, str)
	assert.Error(t, err)

	str, err = registryClient.RegisterService(nil)
	assert.Empty(t, str)
	assert.Error(t, err)

	ms1, err := registryClient.GetProviders("fakeconsumer")
	assert.Empty(t, ms1)
	assert.Error(t, err)

	err = registryClient.AddDependencies(msdepreq)
	assert.Error(t, err)

	err = registryClient.AddDependencies(nil)
	assert.Error(t, err)

	err = registryClient.AddSchemas(MSList[0].ServiceID, "schema", "schema")
	assert.NoError(t, err)

	getms1, err := registryClient.GetMicroService(MSList[0].ServiceID)
	assert.NotEmpty(t, getms1)
	assert.NoError(t, err)

	getms2, err := registryClient.FindMicroServiceInstances("consumerId", MSList[0].AppID, MSList[0].ServiceName, "versionRule", "stage")
	assert.Empty(t, getms2)
	assert.Error(t, err)

	getmsstr, err := registryClient.GetMicroServiceID(MSList[0].AppID, MSList[0].ServiceName, MSList[0].Version)
	assert.NotEmpty(t, getmsstr)
	assert.NoError(t, err)

	getmsstr, err = registryClient.GetMicroServiceID(MSList[0].AppID, "Server112", MSList[0].Version)
	assert.Empty(t, getmsstr)
	assert.Error(t, err)

	ms.Properties = nil
	b, err = registryClient.UpdateMicroServiceProperties(MSList[0].ServiceID, ms)
	assert.Equal(t, false, b)
	assert.Error(t, err)

	err = registryClient.AddSchemas("", "schema", "schema")
	assert.Error(t, err)

	b, err = registryClient.Heartbeat(MSList[0].ServiceID, "")
	assert.Equal(t, false, b)
	assert.Error(t, err)

	b, err = registryClient.UpdateMicroServiceInstanceStatus(MSList[0].ServiceID, "", MSList[0].Status)
	assert.Equal(t, false, b)
	assert.Error(t, err)

	b, err = registryClient.UnregisterMicroService("")
	assert.Equal(t, false, b)
	assert.Error(t, err)
	services, err := registryClient.GetAllResources("instances")
	assert.NotZero(t, len(services))
	assert.NoError(t, err)
	err = registryClient.Close()
	assert.NoError(t, err)

}
