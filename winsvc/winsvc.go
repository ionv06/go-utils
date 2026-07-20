// Package winsvc installs, removes, and runs a Go program as a Windows service.
//
// It is parameterized by Config so the calling application supplies both the
// service identity (name, display name, description) and the lifecycle hooks
// (Run to start the app, Shutdown to stop it gracefully). The Windows-specific
// implementation lives in winsvc_windows.go; other platforms get stubs.
package winsvc

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ionv06/go-utils/utils"
)

// Env var names read by ConfigFromEnv for the service identity.
const (
	EnvServiceName = "SERVICE_NAME"
	EnvDisplayName = "SERVICE_DISPLAY_NAME"
	EnvDescription = "SERVICE_DESCRIPTION"
)

// ErrUnsupported is returned by Install and Remove on non-Windows platforms,
// where there is no Service Control Manager to attach to.
var ErrUnsupported = errUnsupported

// Config describes the service and how to run it.
type Config struct {
	// ServiceName is the key used by the SCM and by `sc`/`services.msc`.
	ServiceName string
	// DisplayName and Description are what the user sees in services.msc.
	DisplayName string
	Description string

	// Run starts the application and blocks until it stops. It is invoked both
	// for console runs and under the SCM.
	Run func() error

	// Shutdown is called when the SCM asks the service to stop. It should stop
	// the application gracefully within the given context deadline. May be nil.
	Shutdown func(ctx context.Context)
}

// ConfigFromEnv loads a .env file (via utils.LoadEnv) and builds a Config from
// the SERVICE_NAME / SERVICE_DISPLAY_NAME / SERVICE_DESCRIPTION variables,
// wiring run and shutdown as the lifecycle hooks. A missing .env is not an error
// here; Install and Remove call Validate to enforce that the identity is set.
func ConfigFromEnv(run func() error, shutdown func(ctx context.Context)) Config {
	_ = utils.LoadEnv()
	return Config{
		ServiceName: strings.TrimSpace(os.Getenv(EnvServiceName)),
		DisplayName: strings.TrimSpace(os.Getenv(EnvDisplayName)),
		Description: strings.TrimSpace(os.Getenv(EnvDescription)),
		Run:         run,
		Shutdown:    shutdown,
	}
}

// Validate errors when any service identity field is empty, naming the missing
// environment variables. Install and Remove call it so they refuse to run
// against an incomplete .env configuration.
func (c Config) Validate() error {
	var missing []string
	if c.ServiceName == "" {
		missing = append(missing, EnvServiceName)
	}
	if c.DisplayName == "" {
		missing = append(missing, EnvDisplayName)
	}
	if c.Description == "" {
		missing = append(missing, EnvDescription)
	}
	if len(missing) > 0 {
		return fmt.Errorf("configurare serviciu incompletă în .env: lipsă %s", strings.Join(missing, ", "))
	}
	return nil
}
