package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// openBrowser öffnet die Anmeldeseite im Standardbrowser. Die Adresse geht
// als ein Argument an das Programm des Systems, nie durch eine Shell, und
// nur als http(s): andere Schemata startet das System als Programm.
func openBrowser(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("die Anmeldeadresse %q ist keine Webadresse", raw)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", raw)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw)
	default:
		cmd = exec.Command("xdg-open", raw)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("der Browser ließ sich nicht öffnen: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
