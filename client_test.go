package sc_test

import (
	"github.com/go-chassis/cari/discovery"
	"github.com/go-chassis/cari/rbac"
	"github.com/go-chassis/openlog"
	"github.com/go-chassis/sc-client"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"
	"time"
)

func TestClient_RegisterService(t *testing.T) {
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	hostname, err := os.Hostname()
	assert.NoError(t, err)

	MSList, err := c.GetAllMicroServices()
	assert.NotEmpty(t, MSList)
	assert.NoError(t, err)

	t.Run("given instance with no service id, should return err", func(t *testing.T) {
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
	})

	t.Run("given wrong service id, should return err", func(t *testing.T) {
		msArr, err := c.GetMicroServiceInstances("fakeConsumerID", "fakeProviderID")
		assert.Empty(t, msArr)
		assert.Error(t, err)

	})

	t.Run("register service with name only", func(t *testing.T) {
		sid, err := c.RegisterService(&discovery.MicroService{
			ServiceName: "simpleService",
		})
		assert.NotEmpty(t, sid)
		s, err := c.GetMicroService(sid)
		assert.NoError(t, err)
		assert.Equal(t, "0.0.1", s.Version)
		assert.Equal(t, "default", s.AppId)
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

}
func TestRegistryClient_FindMicroServiceInstances(t *testing.T) {
	var sid string
	registryClient, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	hostname, err := os.Hostname()
	assert.NoError(t, err)

	ms := &discovery.MicroService{
		ServiceName: "scUTServer",
		AppId:       "default",
		Version:     "0.0.1",
		Schemas:     []string{"schema"},
	}
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
		assert.Equal(t, "{\"schema\":\"schema\"}", string(b))
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

	t.Run("send instance heartbeat (http), should success", func(t *testing.T) {
		iid, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
		assert.NoError(t, err)
		assert.NotNil(t, iid)
		receiveStatus, err := registryClient.Heartbeat(microServiceInstance.ServiceId, iid)
		assert.Equal(t, true, receiveStatus)
		assert.Nil(t, err)
		ok, err := registryClient.UnregisterMicroServiceInstance(microServiceInstance.ServiceId, iid)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("send instance heartbeat (websocket), should success", func(t *testing.T) {
		iid, err := registryClient.RegisterMicroServiceInstance(microServiceInstance)
		assert.NoError(t, err)
		assert.NotNil(t, iid)
		microServiceInstance.InstanceId = iid
		callback := func() {
			registryClient.RegisterMicroServiceInstance(microServiceInstance)
		}
		err = registryClient.WSHeartbeat(microServiceInstance.ServiceId, iid, callback)
		assert.Nil(t, err)
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
func TestClient_Health(t *testing.T) {
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)
	_, err = c.Health()
	assert.NoError(t, err)
}

func TestClient_Auth(t *testing.T) {
	_, err := os.Hostname()
	if err != nil {
		openlog.Error("Get hostname failed.")
		return
	}
	opt := &sc.Options{}
	if !opt.EnableAuth {
		// service-center need to open the rbac module
		return
	}
	// root account login
	c, err := sc.NewClient(
		sc.Options{
			Endpoints:  []string{"127.0.0.1:30100"},
			EnableAuth: true,
			AuthUser: &rbac.AuthUser{
				Username: "root",
				Password: "Complicated_password1",
			},
		})
	assert.NoError(t, err)

	httpHeader := c.GetDefaultHeaders()
	assert.NotEmpty(t, httpHeader)

	t.Run("get the root account token", func(t *testing.T) {
		root_token, err := c.GetToken(&rbac.AuthUser{
			Username: "root",
			Password: "Complicated_password1",
		})
		assert.NoError(t, err)
		assert.NotEmpty(t, root_token)
	})
}
