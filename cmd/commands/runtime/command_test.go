package runtime

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"

	"github.com/urfave/cli/v2"

	runtime_service "github.com/mysteriumnetwork/node/services/runtime/service"
	runtime_capabilities "github.com/mysteriumnetwork/runtime/capabilities"
)

type diagnosticsBackend struct {
	status   runtime_service.RuntimeStatus
	simple   runtime_capabilities.RuntimeCapabilities
	detailed runtime_capabilities.DetailedCapabilities
	dataDir  string
	touched  bool
}

func (backend *diagnosticsBackend) Create(runtime_service.ApprovedCreateOptions) error { return nil }
func (backend *diagnosticsBackend) Delete(string) error                                { return nil }
func (backend *diagnosticsBackend) Start(string) error {
	backend.touched = true
	return nil
}
func (backend *diagnosticsBackend) Stop(string) error {
	backend.touched = true
	return nil
}
func (backend *diagnosticsBackend) SetDesiredState(string, runtime_service.ServiceState) error {
	backend.touched = true
	return nil
}
func (backend *diagnosticsBackend) Get(string) (runtime_service.ServiceInfo, bool, error) {
	return runtime_service.ServiceInfo{}, false, nil
}
func (backend *diagnosticsBackend) DialTCP(string, int) (net.Conn, error) { return nil, nil }
func (backend *diagnosticsBackend) List() ([]runtime_service.ServiceInfo, error) {
	return nil, nil
}
func (backend *diagnosticsBackend) Status() runtime_service.RuntimeStatus { return backend.status }
func (backend *diagnosticsBackend) Availability() runtime_service.Availability {
	return runtime_service.Availability{Available: backend.status.Level != runtime_service.RuntimeLevelUnavailable}
}
func (backend *diagnosticsBackend) Capabilities() (runtime_capabilities.RuntimeCapabilities, runtime_capabilities.DetailedCapabilities) {
	return backend.simple, backend.detailed
}

func TestStatusOutputsCapabilitiesDetailsAndRuntimeJSON(t *testing.T) {
	backend := &diagnosticsBackend{
		status: runtime_service.RuntimeStatus{
			Level:           runtime_service.RuntimeLevelUnavailable,
			BlockingReasons: []string{"mount namespaces are unavailable"},
		},
		simple: runtime_capabilities.RuntimeCapabilities{UserNamespaces: true},
		detailed: runtime_capabilities.DetailedCapabilities{
			UserNamespaces: runtime_capabilities.StatusSupported,
			Seccomp:        runtime_capabilities.StatusNoPerms,
		},
	}
	output := runCommand(t, backend, "node", "runtime", "status")

	var actual statusOutput
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatal(err)
	}
	if !actual.Capabilities.UserNamespaces || actual.Detailed.Seccomp != runtime_capabilities.StatusNoPerms {
		t.Fatalf("unexpected runtime capability report: %#v", actual)
	}
	if actual.Runtime.Level != runtime_service.RuntimeLevelUnavailable || len(actual.Runtime.BlockingReasons) != 1 {
		t.Fatalf("unexpected overall runtime status: %#v", actual.Runtime)
	}
}

// Diagnostics probe the host without starting the node, so inspecting runtime
// status must never reconcile workloads the way a runtime service init does.
func TestStatusDoesNotTouchWorkloads(t *testing.T) {
	backend := &diagnosticsBackend{}
	runCommand(t, backend, "node", "runtime", "status")

	if backend.touched {
		t.Fatal("runtime status started or stopped workloads; diagnostics must not change host state")
	}
}

func TestDataDirIsForwarded(t *testing.T) {
	backend := &diagnosticsBackend{}
	app := cli.NewApp()
	app.Writer = &bytes.Buffer{}
	app.Commands = []*cli.Command{newCommand(func(dataDir string) runtime_service.Backend {
		backend.dataDir = dataDir
		return backend
	})}
	if err := app.Run([]string{"node", "runtime", "--data-dir", "/tmp/runtime-diagnostics", "status"}); err != nil {
		t.Fatal(err)
	}
	if backend.dataDir != "/tmp/runtime-diagnostics" {
		t.Fatalf("data directory was not forwarded: %q", backend.dataDir)
	}
}

func TestOnlyStatusSubcommandIsExposed(t *testing.T) {
	command := newCommand(func(string) runtime_service.Backend { return &diagnosticsBackend{} })
	if len(command.Subcommands) != 1 || command.Subcommands[0].Name != "status" {
		t.Fatalf("unexpected runtime subcommands: %#v", command.Subcommands)
	}
}

func runCommand(t *testing.T, backend runtime_service.Backend, args ...string) []byte {
	t.Helper()
	output := &bytes.Buffer{}
	app := cli.NewApp()
	app.Writer = output
	app.Commands = []*cli.Command{newCommand(func(string) runtime_service.Backend { return backend })}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
