package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser öffnet die Anmeldeseite im Standardbrowser. Die Adresse geht
// als ein Argument an das Programm des Systems, nie durch eine Shell.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("der Browser ließ sich nicht öffnen: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
