/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package endpoints

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	runtime_service "github.com/mysteriumnetwork/node/services/runtime"
	"github.com/mysteriumnetwork/node/services/runtime/registry"
	runtime_service_impl "github.com/mysteriumnetwork/node/services/runtime/service"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

const mockRuntimeServiceType = "runtime-cdp"

var mockRuntimeServiceID = service.ID("6ba7b810-9dad-11d1-80b4-00c04fd430ca")

type mockRuntimeBackend struct {
	services     map[string]runtime_service_impl.ServiceInfo
	status       runtime_service_impl.RuntimeStatus
	availability runtime_service_impl.Availability
	deleted      []string
}

func newMockRuntimeBackend() *mockRuntimeBackend {
	return &mockRuntimeBackend{
		services: map[string]runtime_service_impl.ServiceInfo{
			mockRuntimeServiceType: {
				Name:    mockRuntimeServiceType,
				State:   runtime_service_impl.ServiceStateActive,
				Desired: runtime_service_impl.ServiceStateActive,
				Options: runtime_service_impl.Options{
					Name:        mockRuntimeServiceType,
					OCIArtifact: "registry.example.com/cdp@sha256:abc",
					ServicePort: 9222,
				},
			},
		},
		status:       runtime_service_impl.RuntimeStatus{Level: runtime_service_impl.RuntimeLevelFull},
		availability: runtime_service_impl.Availability{Available: true},
	}
}

func (b *mockRuntimeBackend) Create(options runtime_service_impl.ApprovedCreateOptions) error {
	b.services[options.Name] = runtime_service_impl.ServiceInfo{
		Name:    options.Name,
		State:   runtime_service_impl.ServiceStatePassive,
		Desired: runtime_service_impl.ServiceStatePassive,
		Options: runtime_service_impl.Options{Name: options.Name},
	}
	return nil
}

func (b *mockRuntimeBackend) Delete(name string) error {
	b.deleted = append(b.deleted, name)
	delete(b.services, name)
	return nil
}

func (b *mockRuntimeBackend) Start(name string) error { return nil }

func (b *mockRuntimeBackend) Stop(name string) error { return nil }

func (b *mockRuntimeBackend) SetDesiredState(name string, state runtime_service_impl.ServiceState) error {
	return nil
}

func (b *mockRuntimeBackend) Get(name string) (runtime_service_impl.ServiceInfo, bool, error) {
	info, exists := b.services[name]
	return info, exists, nil
}

func (b *mockRuntimeBackend) DialTCP(name string, port int) (net.Conn, error) { return nil, nil }

func (b *mockRuntimeBackend) List() ([]runtime_service_impl.ServiceInfo, error) {
	list := make([]runtime_service_impl.ServiceInfo, 0, len(b.services))
	for _, info := range b.services {
		list = append(list, info)
	}
	return list, nil
}

func (b *mockRuntimeBackend) Status() runtime_service_impl.RuntimeStatus { return b.status }

func (b *mockRuntimeBackend) Availability() runtime_service_impl.Availability { return b.availability }

func (b *mockRuntimeBackend) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return runtime_capabilities.RuntimeCapabilities{}, runtime_capabilities.DetailedCapabilities{}
}

type mockRuntimeInstaller struct {
	installed []runtime_service_impl.CreateOptions
	err       error
	backend   *mockRuntimeBackend
}

func (i *mockRuntimeInstaller) Install(options runtime_service_impl.CreateOptions) error {
	if i.err != nil {
		return i.err
	}
	i.installed = append(i.installed, options)
	if i.backend != nil {
		return i.backend.Create(runtime_service_impl.ApprovedCreateOptions{CreateOptions: options})
	}
	return nil
}

// mockRuntimeServiceManager reports the node-side service instances, which is
// how the endpoint decides whether the parent runtime service is running and
// which instance a delete has to stop first.
type mockRuntimeServiceManager struct {
	instances []*service.Instance
	stopped   []service.ID
	stopErr   error
}

func (sm *mockRuntimeServiceManager) Start(_ identity.Identity, _ string, _ []string, _ service.Options) (service.ID, error) {
	return mockRuntimeServiceID, nil
}

func (sm *mockRuntimeServiceManager) Stop(id service.ID) error {
	if sm.stopErr != nil {
		return sm.stopErr
	}
	sm.stopped = append(sm.stopped, id)
	return nil
}

func (sm *mockRuntimeServiceManager) Service(id service.ID) *service.Instance {
	for _, instance := range sm.instances {
		if instance.ID == id {
			return instance
		}
	}
	return nil
}

func (sm *mockRuntimeServiceManager) List(includeAll bool) []*service.Instance { return sm.instances }

func (sm *mockRuntimeServiceManager) Kill() error { return nil }

func runtimeInstance(id service.ID, serviceType string) *service.Instance {
	instance := service.NewInstance(
		mockProviderID,
		serviceType,
		nil,
		market.NewProposal(mockProviderID.Address, serviceType, market.NewProposalOpts{}),
		servicestate.Running,
		nil, nil, nil,
	)
	instance.ID = id
	return instance
}

// activeRuntimeManager has the parent runtime service running, which every
// managing call requires.
func activeRuntimeManager(extra ...*service.Instance) *mockRuntimeServiceManager {
	instances := []*service.Instance{runtimeInstance("runtime-parent-id", runtime_service.ServiceType)}
	return &mockRuntimeServiceManager{instances: append(instances, extra...)}
}

func serveRuntime(
	t *testing.T,
	backend runtime_service_impl.Backend,
	installer runtime_service_impl.Installer,
	serviceManager ServiceManager,
	req *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	resp := httptest.NewRecorder()
	g := summonTestGin()
	err := AddRoutesForRuntime(backend, installer, serviceManager)(g)
	assert.NoError(t, err)
	g.ServeHTTP(resp, req)
	return resp
}

func Test_RuntimeStatus_ReportsAvailability(t *testing.T) {
	backend := newMockRuntimeBackend()
	backend.availability = runtime_service_impl.Availability{Available: false, Reason: "user namespaces are unavailable"}

	resp := serveRuntime(t, backend, &mockRuntimeInstaller{}, activeRuntimeManager(),
		httptest.NewRequest(http.MethodGet, "/runtime/status", nil))

	assert.Equal(t, http.StatusOK, resp.Code)

	var status contract.RuntimeStatusDTO
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &status))
	assert.False(t, status.Available)
	assert.Equal(t, "user namespaces are unavailable", status.Reason)
	assert.Equal(t, runtime_service_impl.RuntimeLevelFull, status.Runtime.Level)
}

func Test_Runtime_WithoutBackendIsUnavailable(t *testing.T) {
	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/runtime/status", nil),
		httptest.NewRequest(http.MethodGet, "/runtime/services", nil),
		httptest.NewRequest(http.MethodGet, "/runtime/services/cdp", nil),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"cdp"}`)),
		httptest.NewRequest(http.MethodDelete, "/runtime/services/cdp", nil),
	} {
		resp := serveRuntime(t, nil, nil, activeRuntimeManager(), req)

		assert.Equal(t, http.StatusServiceUnavailable, resp.Code, req.Method+" "+req.URL.Path)
		assert.Equal(t, contract.ErrCodeRuntimeNotConfigured, apierror.Parse(resp.Result()).Err.Code)
	}
}

func Test_RuntimeServiceList_ReportsInstalledAndRunning(t *testing.T) {
	backend := newMockRuntimeBackend()
	backend.services["runtime-idle"] = runtime_service_impl.ServiceInfo{
		Name:    "runtime-idle",
		State:   runtime_service_impl.ServiceStatePassive,
		Desired: runtime_service_impl.ServiceStatePassive,
	}
	manager := activeRuntimeManager(runtimeInstance(mockRuntimeServiceID, mockRuntimeServiceType))

	resp := serveRuntime(t, backend, &mockRuntimeInstaller{}, manager,
		httptest.NewRequest(http.MethodGet, "/runtime/services", nil))

	assert.Equal(t, http.StatusOK, resp.Code)

	var list contract.RuntimeServiceListResponse
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &list))
	assert.Len(t, list.Services, 2)

	byName := map[string]contract.RuntimeServiceInfoDTO{}
	for _, s := range list.Services {
		byName[s.Name] = s
	}
	// A running definition carries the instance ID needed to stop it.
	assert.Equal(t, string(mockRuntimeServiceID), byName[mockRuntimeServiceType].ServiceID)
	// An installed but not running one is listed all the same, which is what
	// GET /services cannot report.
	assert.Empty(t, byName["runtime-idle"].ServiceID)
	assert.Equal(t, runtime_service_impl.ServiceStatePassive, byName["runtime-idle"].DesiredState)
}

func Test_RuntimeServiceGet_NormalizesName(t *testing.T) {
	for _, name := range []string{"cdp", "runtime-cdp", "CDP"} {
		resp := serveRuntime(t, newMockRuntimeBackend(), &mockRuntimeInstaller{}, activeRuntimeManager(),
			httptest.NewRequest(http.MethodGet, "/runtime/services/"+name, nil))

		assert.Equal(t, http.StatusOK, resp.Code, name)

		var info contract.RuntimeServiceInfoDTO
		assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &info))
		assert.Equal(t, mockRuntimeServiceType, info.Name)
		assert.Equal(t, 9222, info.Options.ServicePort)
	}
}

func Test_RuntimeServiceGet_NotInstalled(t *testing.T) {
	resp := serveRuntime(t, newMockRuntimeBackend(), &mockRuntimeInstaller{}, activeRuntimeManager(),
		httptest.NewRequest(http.MethodGet, "/runtime/services/absent", nil))

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func Test_RuntimeServiceInstall_InstallsNormalizedName(t *testing.T) {
	backend := newMockRuntimeBackend()
	installer := &mockRuntimeInstaller{backend: backend}

	resp := serveRuntime(t, backend, installer, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services",
			strings.NewReader(`{"name":"scraper","minimum_runtime_level":"limited"}`)))

	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.Len(t, installer.installed, 1)
	assert.Equal(t, "runtime-scraper", installer.installed[0].Name)
	assert.Equal(t, runtime_service_impl.RuntimeLevelLimited, installer.installed[0].MinimumRuntimeLevel)

	var info contract.RuntimeServiceInfoDTO
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &info))
	assert.Equal(t, "runtime-scraper", info.Name)
	// Installing does not start anything.
	assert.Empty(t, info.ServiceID)
}

func Test_RuntimeServiceInstall_RejectsArtifactSelection(t *testing.T) {
	installer := &mockRuntimeInstaller{}

	resp := serveRuntime(t, newMockRuntimeBackend(), installer, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services",
			strings.NewReader(`{"name":"cdp","oci_artifact":"evil.example.com/x@sha256:def"}`)))

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Empty(t, installer.installed)
}

func Test_RuntimeServiceInstall_RequiresName(t *testing.T) {
	resp := serveRuntime(t, newMockRuntimeBackend(), &mockRuntimeInstaller{}, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"  "}`)))

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	apiErr := apierror.Parse(resp.Result())
	assert.Contains(t, apiErr.Err.Fields, "name")
}

func Test_RuntimeServiceInstall_UnlistedServiceIsRefused(t *testing.T) {
	installer := &mockRuntimeInstaller{err: errors.Wrap(registry.ErrNotListed, `runtime service "runtime-evil"`)}

	resp := serveRuntime(t, newMockRuntimeBackend(), installer, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"evil"}`)))

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.Equal(t, contract.ErrCodeRuntimeNotListed, apierror.Parse(resp.Result()).Err.Code)
}

func Test_RuntimeServiceInstall_WithoutInstallerIsUnavailable(t *testing.T) {
	resp := serveRuntime(t, newMockRuntimeBackend(), nil, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"cdp"}`)))

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, contract.ErrCodeRuntimeNotConfigured, apierror.Parse(resp.Result()).Err.Code)
}

func Test_RuntimeServiceInstall_RequiresAvailableRuntime(t *testing.T) {
	backend := newMockRuntimeBackend()
	backend.status = runtime_service_impl.RuntimeStatus{
		Level:           runtime_service_impl.RuntimeLevelUnavailable,
		BlockingReasons: []string{"user namespaces are unavailable"},
	}
	installer := &mockRuntimeInstaller{}

	resp := serveRuntime(t, backend, installer, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"cdp"}`)))

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.Equal(t, contract.ErrCodeRuntimeUnavailable, apierror.Parse(resp.Result()).Err.Code)
	assert.Empty(t, installer.installed)
}

func Test_RuntimeServiceInstall_RequiresRunningRuntimeService(t *testing.T) {
	installer := &mockRuntimeInstaller{}

	resp := serveRuntime(t, newMockRuntimeBackend(), installer, &mockRuntimeServiceManager{},
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"cdp"}`)))

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	assert.Equal(t, contract.ErrCodeRuntimeInactive, apierror.Parse(resp.Result()).Err.Code)
	assert.Empty(t, installer.installed)
}

func Test_RuntimeServiceDelete_StopsRunningServiceFirst(t *testing.T) {
	backend := newMockRuntimeBackend()
	manager := activeRuntimeManager(runtimeInstance(mockRuntimeServiceID, mockRuntimeServiceType))

	resp := serveRuntime(t, backend, &mockRuntimeInstaller{}, manager,
		httptest.NewRequest(http.MethodDelete, "/runtime/services/cdp", nil))

	assert.Equal(t, http.StatusAccepted, resp.Code)
	assert.Equal(t, []service.ID{mockRuntimeServiceID}, manager.stopped)
	assert.Equal(t, []string{mockRuntimeServiceType}, backend.deleted)
}

func Test_RuntimeServiceDelete_KeepsDefinitionWhenStopFails(t *testing.T) {
	backend := newMockRuntimeBackend()
	manager := activeRuntimeManager(runtimeInstance(mockRuntimeServiceID, mockRuntimeServiceType))
	manager.stopErr = errors.New("stop failed")

	resp := serveRuntime(t, backend, &mockRuntimeInstaller{}, manager,
		httptest.NewRequest(http.MethodDelete, "/runtime/services/cdp", nil))

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.Empty(t, backend.deleted)
}

func Test_RuntimeServiceDelete_NotInstalled(t *testing.T) {
	backend := newMockRuntimeBackend()

	resp := serveRuntime(t, backend, &mockRuntimeInstaller{}, activeRuntimeManager(),
		httptest.NewRequest(http.MethodDelete, "/runtime/services/absent", nil))

	assert.Equal(t, http.StatusNotFound, resp.Code)
	assert.Empty(t, backend.deleted)
}

func Test_RuntimeServiceInstall_UnconfiguredRegistryIsUnavailable(t *testing.T) {
	// A real installer with no registry behind it: the node cannot admit any
	// workload, which is a misconfiguration rather than a bad request.
	installer := runtime_service_impl.NewInstaller(newMockRuntimeBackend(), nil)

	resp := serveRuntime(t, newMockRuntimeBackend(), installer, activeRuntimeManager(),
		httptest.NewRequest(http.MethodPost, "/runtime/services", strings.NewReader(`{"name":"cdp"}`)))

	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)
	assert.Equal(t, contract.ErrCodeRuntimeNotConfigured, apierror.Parse(resp.Result()).Err.Code)
}
