package sc_test

import (
	"github.com/go-chassis/cari/discovery"
	"github.com/go-chassis/sc-client"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/go-chassis/openlog"
	"os"
	"time"
)

func TestNewClient(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		openlog.Error("Get hostname failed.")
		return
	}
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	err = c.SyncEndpoints()
	assert.NoError(t, err)

	httpHeader := c.GetDefaultHeaders()
	assert.NotEmpty(t, httpHeader)

	MSList, err := c.GetAllMicroServices()
	assert.NotEmpty(t, MSList)
	assert.NoError(t, err)

	f1 := func(*sc.MicroServiceInstanceChangedEvent) {}
	err = c.WatchMicroService(MSList[0].ServiceId, f1)
	assert.NoError(t, err)

	var ms = new(discovery.MicroService)
	var m = make(map[string]string)

	m["abc"] = "abc"
	m["def"] = "def"

	ms.AppId = MSList[0].AppId
	ms.ServiceName = MSList[0].ServiceName
	ms.Version = MSList[0].Version
	ms.Environment = MSList[0].Environment
	ms.Properties = m

	microServiceInstance := &discovery.MicroServiceInstance{
		Endpoints: []string{"rest://127.0.0.1:3000"},
		HostName:  hostname,
		Status:    sc.MSInstanceUP,
	}
	s1, err := c.RegisterMicroServiceInstance(microServiceInstance)
	assert.Empty(t, s1)
	assert.Error(t, err)

	s1, err = c.RegisterMicroServiceInstance(nil)
	assert.Empty(t, s1)
	assert.Error(t, err)

	msArr, err := c.GetMicroServiceInstances("fakeConsumerID", "fakeProviderID")
	assert.Empty(t, msArr)
	assert.Error(t, err)

	msArr, err = c.Health()
	assert.NotEmpty(t, msArr)
	assert.NoError(t, err)

	b, err := c.UpdateMicroServiceProperties(MSList[0].ServiceId, ms)
	assert.Equal(t, true, b)
	assert.NoError(t, err)

	f1 = func(*sc.MicroServiceInstanceChangedEvent) {}
	err = c.WatchMicroService(MSList[0].ServiceId, f1)
	assert.NoError(t, err)

	f1 = func(*sc.MicroServiceInstanceChangedEvent) {}
	err = c.WatchMicroService("", f1)
	assert.Error(t, err)

	f1 = func(*sc.MicroServiceInstanceChangedEvent) {}
	err = c.WatchMicroService(MSList[0].ServiceId, nil)
	assert.NoError(t, err)

	str, err := c.RegisterService(ms)
	assert.NotEmpty(t, str)
	assert.NoError(t, err)

	str, err = c.RegisterService(nil)
	assert.Empty(t, str)
	assert.Error(t, err)
	t.Run("register service with name only", func(t *testing.T) {
		sid, err := c.RegisterService(&discovery.MicroService{
			ServiceName: "simpleService",
		})
		assert.NotEmpty(t, sid)
		assert.NoError(t, err)
		s, err := c.GetMicroService(sid)
		assert.NoError(t, err)
		assert.Equal(t, "0.0.1", s.Version)
		assert.Equal(t, "default", s.AppId)
		ok, err := c.UnregisterMicroService(sid)
		assert.NoError(t, err)
		assert.True(t, ok)
		s, err = c.GetMicroService(sid)
		assert.Nil(t, s)
	})
	t.Run("register service with invalid name", func(t *testing.T) {
		_, err := c.RegisterService(&discovery.MicroService{
			ServiceName: "simple&Service",
		})
		t.Log(err)
		assert.Error(t, err)
	})
	t.Run("get all apps, not empty", func(t *testing.T) {
		apps, err := c.GetAllApplications()
		assert.NoError(t, err)
		assert.NotEqual(t, 0, len(apps))
		t.Log(len(apps))

	})
	ms1, err := c.GetProviders("fakeconsumer")
	assert.Empty(t, ms1)
	assert.Error(t, err)

	getms1, err := c.GetMicroService(MSList[0].ServiceId)
	assert.NotEmpty(t, getms1)
	assert.NoError(t, err)

	getms2, err := c.FindMicroServiceInstances("abcd", MSList[0].AppId, MSList[0].ServiceName, MSList[0].Version)
	assert.Empty(t, getms2)
	assert.Error(t, err)

	getmsstr, err := c.GetMicroServiceID(MSList[0].AppId, MSList[0].ServiceName, MSList[0].Version, MSList[0].Environment)
	assert.NotEmpty(t, getmsstr)
	assert.NoError(t, err)

	getmsstr, err = c.GetMicroServiceID(MSList[0].AppId, "Server112", MSList[0].Version, "")
	assert.Empty(t, getmsstr)
	//assert.Error(t, err)

	ms.Properties = nil
	b, err = c.UpdateMicroServiceProperties(MSList[0].ServiceId, ms)
	assert.Equal(t, false, b)
	assert.Error(t, err)

	err = c.AddSchemas("", "schema", "schema")
	assert.Error(t, err)

	b, err = c.Heartbeat(MSList[0].ServiceId, "")
	assert.Equal(t, false, b)
	assert.Error(t, err)

	b, err = c.UpdateMicroServiceInstanceStatus(MSList[0].ServiceId, "", MSList[0].Status)
	assert.Equal(t, false, b)
	assert.Error(t, err)

	b, err = c.UnregisterMicroService("")
	assert.Equal(t, false, b)
	assert.Error(t, err)
	services, err := c.GetAllResources("instances")
	assert.NotZero(t, len(services))
	assert.NoError(t, err)
	err = c.Close()
	assert.NoError(t, err)

}
func TestRegistryClient_FindMicroServiceInstances(t *testing.T) {

	hostname, err := os.Hostname()
	if err != nil {
		openlog.Error("Get hostname failed.")
		return
	}
	ms := &discovery.MicroService{
		ServiceName: "scUTServer",
		AppId:       "default",
		Version:     "0.0.1",
		Schemas:     []string{"schema"},
	}
	var sid string

	registryClient, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)
	sid, err = registryClient.RegisterService(ms)

	if err == sc.ErrMicroServiceExists {
		sid, err = registryClient.GetMicroServiceID("default", "scUTServer", "0.0.1", "")
		assert.NoError(t, err)
		assert.NotNil(t, sid)
	}

	err = registryClient.AddSchemas(ms.ServiceId, "schema", "schema")
	assert.NoError(t, err)
	t.Run("query schema, should return info", func(t *testing.T) {
		b, err := registryClient.GetSchema(ms.ServiceId, "schema")
		assert.NoError(t, err)
		assert.Equal(t, "{\"schema\":\"schema\"}\n", string(b))
	})
	t.Run("query schema with empty string, should be err", func(t *testing.T) {
		_, err := registryClient.GetSchema("", "schema")
		assert.Error(t, err)
	})
	microServiceInstance := &discovery.MicroServiceInstance{
		ServiceId: sid,
		Endpoints: []string{"rest://127.0.0.1:3000"},
		HostName:  hostname,
		Status:    sc.MSInstanceUP,
	}
	t.Run("unregister instance, should success", func(t *testing.T) {
		iid, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
		assert.NoError(t, err)
		assert.NotNil(t, iid)
		ok, err := registryClient.UnregisterMicroServiceInstance(microServiceInstance.ServiceId, iid)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("register instance and update props, should success", func(t *testing.T) {
		iid, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
		assert.NoError(t, err)
		assert.NotNil(t, iid)
		microServiceInstance.Properties = map[string]string{
			"project": "x"}
		ok, err := registryClient.UpdateMicroServiceInstanceProperties(microServiceInstance.ServiceId,
			iid, microServiceInstance)
		assert.True(t, ok)
		assert.NoError(t, err)
		instances, err := registryClient.FindMicroServiceInstances(microServiceInstance.ServiceId,
			"default",
			"scUTServer", "0.0.1")
		assert.NoError(t, err)
		assert.Equal(t, "x", instances[0].Properties["project"])
	})

	t.Log("find again, should get ErrNotModified")
	_, err = registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0.0.1")
	assert.Equal(t, sc.ErrNotModified, err)

	t.Log("find again without revision, should get nil error")
	_, err = registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0.0.1",
		sc.WithoutRevision())
	assert.NoError(t, err)

	t.Log("register new and find")
	microServiceInstance2 := &discovery.MicroServiceInstance{
		ServiceId: sid,
		Endpoints: []string{"rest://127.0.0.1:3001"},
		HostName:  hostname + "1",
		Status:    sc.MSInstanceUP,
	}
	_, err = registryClient.RegisterMicroServiceInstance(microServiceInstance2)
	time.Sleep(3 * time.Second)
	_, err = registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0.0.1")
	assert.NoError(t, err)

	t.Log("after reset")
	registryClient.ResetRevision()
	_, err = registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0.0.1")
	assert.NoError(t, err)
	_, err = registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0.0.1")
	assert.Equal(t, sc.ErrNotModified, err)

	_, err = registryClient.FindMicroServiceInstances(sid, "AppIdNotExists", "ServerNotExists", "0.0.1")
	assert.Equal(t, sc.ErrMicroServiceNotExists, err)

	f := &discovery.FindService{
		Service: &discovery.MicroServiceKey{
			ServiceName: "scUTServer",
			AppId:       "default",
			Version:     "0.0.1",
		},
	}
	fs := []*discovery.FindService{f}
	instances, err := registryClient.BatchFindInstances(sid, fs)
	t.Log(instances)
	assert.NoError(t, err)

	f1 := &discovery.FindService{
		Service: &discovery.MicroServiceKey{
			ServiceName: "empty",
			AppId:       "default",
			Version:     "0.0.1",
		},
	}
	fs = []*discovery.FindService{f1}
	instances, err = registryClient.BatchFindInstances(sid, fs)
	t.Log(instances)
	assert.NoError(t, err)

	f2 := &discovery.FindService{
		Service: &discovery.MicroServiceKey{
			ServiceName: "empty",
			AppId:       "default",
			Version:     "latest",
		},
	}
	fs = []*discovery.FindService{f}
	instances, err = registryClient.BatchFindInstances(sid, fs)
	t.Log(instances)
	assert.NoError(t, err)

	fs = []*discovery.FindService{f2, f}
	instances, err = registryClient.BatchFindInstances(sid, fs)
	t.Log(instances)
	assert.NoError(t, err)

	fs = []*discovery.FindService{}
	instances, err = registryClient.BatchFindInstances(sid, fs)
	assert.Equal(t, sc.ErrEmptyCriteria, err)
}
