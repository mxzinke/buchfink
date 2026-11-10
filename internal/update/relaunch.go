package update

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// predecessorFlag nennt einem frisch gestarteten Buchfink den Prozess, den es
// ablöst. Buchfink läuft nur einmal je Rechner (Einzelinstanz, Sperre des
// Datenordners, Port des MCP-Servers); der Nachfolger wartet deshalb, bis der
// Vorgänger beendet ist.
const predecessorFlag = "--abloesung-von="

// predecessorTimeout begrenzt das Warten. Der Vorgänger sichert beim Beenden
// und schließt den Mandanten; danach startet der Nachfolger in jedem Fall.
const predecessorTimeout = 60 * time.Second

// Relaunch startet die installierte Fassung als eigenen Prozess. Der Aufrufer
// beendet danach die laufende App.
func (s *Service) Relaunch() error {
	if s.State().Phase != PhaseInstalled {
		return fmt.Errorf("es ist keine neue Fassung installiert")
	}
	installation, err := s.locate()
	if err != nil {
		return err
	}
	cmd := relaunchCommand(installation, os.Getpid())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("die neue Fassung ließ sich nicht starten; öffnen Sie Buchfink bitte selbst: %w", err)
	}
	return cmd.Process.Release()
}

func relaunchCommand(installation Installation, pid int) *exec.Cmd {
	flag := predecessorFlag + strconv.Itoa(pid)
	if installation.Bundle {
		// open startet das Paket über den Dienst des Systems, wie ein
		// Doppelklick; -n erlaubt den Start, solange der Vorgänger noch läuft.
		return exec.Command("/usr/bin/open", "-n", installation.Path, "--args", flag)
	}
	cmd := exec.Command(installation.Path, flag)
	detach(cmd)
	return cmd
}

// WaitForPredecessor wartet beim Start nach einer Aktualisierung, bis der
// abgelöste Prozess beendet ist. Ohne die Angabe kehrt es sofort zurück.
func WaitForPredecessor(args []string) {
	for _, arg := range args {
		value, ok := strings.CutPrefix(arg, predecessorFlag)
		if !ok {
			continue
		}
		if pid, err := strconv.Atoi(value); err == nil && pid > 0 {
			waitForExit(pid, predecessorTimeout)
		}
		return
	}
}
