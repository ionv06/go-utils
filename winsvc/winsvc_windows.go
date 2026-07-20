//go:build windows

package winsvc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// errUnsupported is never returned on Windows; it exists so the shared
// ErrUnsupported symbol compiles on all platforms.
var errUnsupported = errors.New("gestionarea serviciului este disponibilă doar pe Windows")

// Run runs the app under the Windows Service Control Manager when the process
// was started as a service; otherwise it runs as a normal console app.
func Run(cfg Config) error {
	isSvc, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("detecție mod serviciu: %w", err)
	}
	if !isSvc {
		return cfg.Run()
	}

	// The SCM starts services in %WinDir%\System32, so relative paths (the .env
	// config next to the binary) would not resolve. Switch to the executable's
	// directory before starting the app.
	if exe, err := os.Executable(); err == nil {
		_ = os.Chdir(filepath.Dir(exe))
	}

	return svc.Run(cfg.ServiceName, &handler{cfg: cfg})
}

// handler adapts cfg.Run to the svc.Handler interface expected by svc.Run.
type handler struct{ cfg Config }

// Execute is called by the SCM. It launches the app in a goroutine and then
// services control requests (stop/shutdown) until told to exit. An app that
// exits on its own reports a non-zero code so the SCM's recovery actions
// (configured at install time) restart it.
func (h *handler) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (svcSpecificEC bool, exitCode uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}

	errCh := make(chan error, 1)
	go func() { errCh <- h.cfg.Run() }()

	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				if h.cfg.Shutdown != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					h.cfg.Shutdown(ctx)
					cancel()
				}
				return false, 0
			default:
				// Ignore unexpected control codes.
			}
		case err := <-errCh:
			// The app stopped by itself. Report failure on error so the SCM
			// triggers recovery; a clean exit ends the service normally.
			changes <- svc.Status{State: svc.StopPending}
			if err != nil {
				return false, 1
			}
			return false, 0
		}
	}
}

// Install registers the executable as an auto-start Windows service with
// automatic recovery (restart on failure). Must be run from an elevated prompt.
func Install(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cale executabil: %w", err)
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return fmt.Errorf("cale absolută executabil: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("conectare la Service Control Manager (rulează ca administrator): %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(cfg.ServiceName); err == nil {
		s.Close()
		return fmt.Errorf("serviciul %q există deja", cfg.ServiceName)
	}

	s, err := m.CreateService(cfg.ServiceName, exe, mgr.Config{
		DisplayName:  cfg.DisplayName,
		Description:  cfg.Description,
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
	})
	if err != nil {
		return fmt.Errorf("creare serviciu: %w", err)
	}
	defer s.Close()

	// Recovery: restart the service on each of the first failures, then keep
	// restarting; reset the failure counter once a day.
	recovery := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
	}
	if err := s.SetRecoveryActions(recovery, uint32((24 * time.Hour).Seconds())); err != nil {
		// The service is installed; surface the recovery-config failure but leave
		// the service in place so the caller can retry or fix it manually.
		return fmt.Errorf("serviciu instalat, dar configurarea recuperării a eșuat: %w", err)
	}

	return nil
}

// Remove stops (if running) and deletes the Windows service. Must be run from
// an elevated prompt.
func Remove(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("conectare la Service Control Manager (rulează ca administrator): %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(cfg.ServiceName)
	if err != nil {
		return fmt.Errorf("serviciul %q nu este instalat", cfg.ServiceName)
	}
	defer s.Close()

	// Best-effort stop before delete; ignore errors (may already be stopped).
	_, _ = s.Control(svc.Stop)

	if err := s.Delete(); err != nil {
		return fmt.Errorf("ștergere serviciu: %w", err)
	}
	return nil
}
