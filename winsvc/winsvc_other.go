//go:build !windows

package winsvc

import "errors"

// errUnsupported backs ErrUnsupported on non-Windows platforms, where the app
// is deployed as a Linux binary rather than a Windows service.
var errUnsupported = errors.New("gestionarea serviciului este disponibilă doar pe Windows")

// Run simply runs the app; there is no Service Control Manager to attach to.
func Run(cfg Config) error { return cfg.Run() }

// Install is unsupported off Windows.
func Install(cfg Config) error { return errUnsupported }

// Remove is unsupported off Windows.
func Remove(cfg Config) error { return errUnsupported }
