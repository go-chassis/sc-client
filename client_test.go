package sc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-chassis/cari/discovery"
	"github.com/go-chassis/cari/rbac"
	"github.com/go-chassis/openlog"
	"github.com/stretchr/testify/assert"

	"github.com/go-chassis/sc-client"
)

func TestClient_RegisterService(t *testing.T) {
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	hostname, err := os.Hostname()
	assert.NoError(t, err)

	_, err = c.GetAllMicroServices()
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

	var rst *sc.FindMicroServiceInstancesResult
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
		rst, err = registryClient.FindInstances(microServiceInstance.ServiceId,
			"default",
			"scUTServer")
		assert.NoError(t, err)
		assert.Equal(t, "x", rst.Instances[0].Properties["project"])
	})

	t.Log("find again, should get no error")
	rstInstances, err := registryClient.FindMicroServiceInstances(sid, "default", "scUTServer", "0%2B")
	assert.NoError(t, err)
	assert.NotEmpty(t, rstInstances)

	t.Log("find again, should get no error")
	newRst, err := registryClient.FindInstances(sid, "default", "scUTServer")
	assert.NoError(t, err)
	assert.NotEmpty(t, newRst.Revision)
	assert.NotEmpty(t, newRst.Instances)
	assert.Equal(t, newRst.Revision, rst.Revision)

	t.Log("find again without revision, should get nil error")
	newRst, err = registryClient.FindInstances(sid, "default", "scUTServer", sc.WithoutRevision())
	assert.NoError(t, err)
	assert.NotEmpty(t, newRst.Revision)
	assert.NotEmpty(t, newRst.Instances)
	assert.Equal(t, newRst.Revision, rst.Revision)

	t.Log("find again with different revision, should get nil error")
	newRst, err = registryClient.FindInstances(sid, "default", "scUTServer", sc.WithRevision("123"))
	assert.NoError(t, err)
	assert.NotEmpty(t, newRst.Revision)
	assert.NotEmpty(t, newRst.Instances)

	t.Log("find again with same revision, should get ErrNotModified")
	newRst, err = registryClient.FindInstances(sid, "default", "scUTServer", sc.WithRevision(newRst.Revision))
	assert.Equal(t, sc.ErrNotModified, err)

	t.Log("register new and find")
	microServiceInstance2 := &discovery.MicroServiceInstance{
		ServiceId: sid,
		Endpoints: []string{"rest://127.0.0.1:3001"},
		HostName:  hostname + "1",
		Status:    sc.MSInstanceUP,
	}
	_, err = registryClient.RegisterMicroServiceInstance(microServiceInstance2)
	time.Sleep(3 * time.Second)
	_, err = registryClient.FindInstances(sid, "default", "scUTServer")
	assert.NoError(t, err)

	_, err = registryClient.FindInstances(sid, "AppIdNotExists", "ServerNotExists")
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

func TestClient_CheckPeerStatus(t *testing.T) {
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)
	_, err = c.CheckPeerStatus()
	assert.Equal(t, "Common exception", err.(*sc.RegistryException).Title)
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

func TestClient_DataRace(t *testing.T) {
	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{"127.0.0.1:30100"},
		})
	assert.NoError(t, err)

	MSList, err := c.GetAllMicroServices()
	assert.NotEmpty(t, MSList)
	assert.NoError(t, err)

	t.Run("should not race detected", func(t *testing.T) {
		wg := sync.WaitGroup{}
		wg.Add(2)
		go func() {
			sc.NewClient(
				sc.Options{
					Endpoints: []string{"127.0.0.1:30100"},
				})
			wg.Done()
		}()

		go func() {
			c.GetAllMicroServices()
			wg.Done()
		}()

		wg.Wait()
	})
}

func TestClient_SyncEndpoints(t *testing.T) {
	os.Setenv("CHASSIS_SC_HEALTH_CHECK_INTERVAL", "1")

	anotherScServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		return
	}))

	scServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		resp := &discovery.GetInstancesResponse{
			Instances: []*discovery.MicroServiceInstance{
				{
					Endpoints: []string{"rest://" + anotherScServer.Listener.Addr().String()},
					HostName:  "test",
					Status:    sc.MSInstanceUP,
					DataCenterInfo: &discovery.DataCenterInfo{
						Name:          "engine1",
						Region:        "cn",
						AvailableZone: "az1",
					},
				},
			},
		}
		instanceBytes, err := json.Marshal(resp)
		if err != nil {
			writer.Write([]byte(err.Error()))
			writer.WriteHeader(http.StatusInternalServerError)
		}
		writer.Write(instanceBytes)
		writer.WriteHeader(http.StatusOK)
		return
	}))

	c, err := sc.NewClient(
		sc.Options{
			Endpoints: []string{scServer.Listener.Addr().String()},
		})
	assert.NoError(t, err)
	assert.Equal(t, scServer.Listener.Addr().String(), c.GetAddress()) // default

	err = c.SyncEndpoints()
	assert.Equal(t, scServer.Listener.Addr().String(), c.GetAddress())

	scServer.Close()
	time.Sleep(3*time.Second + 100*time.Millisecond)
	// sc stopped, should use the synced address
	assert.Equal(t, anotherScServer.Listener.Addr().String(), c.GetAddress())
}
