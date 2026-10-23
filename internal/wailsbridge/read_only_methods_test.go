package wailsbridge

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Was als lesend eingeordnet ist, schreibt nicht.
//
// Ein Lesevorgang läuft auf einer Verbindung mit `query_only`; ein
// Schreibversuch scheitert dort mit „readonly". Der Test ruft jede Methode der
// Leseliste mit Nullwerten auf einem Mandanten mit Bestand auf. Nullwerte
// erreichen nicht jeden Pfad, aber jeden, der ohne Eingabe läuft — und
// genau dort steht eine vergessene Reparatur oder ein Zwischenspeicher.
func TestReadOnlyMethodsDoNotWrite(t *testing.T) {
	b := testBridge(t)
	request := manualEntryRequest(t, b)
	entry, err := postManualEntry(b, request)
	if err != nil {
		t.Fatalf("Bestand anlegen: %v", err)
	}

	// Mandantenverwaltung, Prüfermodus und Sicherung schreiben bewusst; sie
	// stehen in der Leseliste, weil sie im Prüfermodus zulässig sind.
	skip := map[string]bool{
		"SetupApplication": true, "LoadExistingDatabase": true,
		"RestoreFromBackup": true, "SwitchTenant": true, "DeleteTenant": true,
		"CreateBackup": true, "SetBackupDir": true, "EnableReadOnly": true,
		"DisableReadOnly": true, "ServiceShutdown": true, "SetFiscalYear": true,
		"OpenReleasesPage": true, "OpenAgentGuide": true,
	}

	// Zwei Durchgänge: mit Nullwerten und mit einem Jahr, einer Kennung und
	// einem Stichtag, die es im Bestand gibt.
	bridge := reflect.ValueOf(b)
	called := 0
	for _, filled := range []bool{false, true} {
		for name := range readOnlyAllowed {
			if skip[name] {
				continue
			}
			method := bridge.MethodByName(name)
			if !method.IsValid() {
				continue
			}
			args := make([]reflect.Value, method.Type().NumIn())
			for i := range args {
				args[i] = argument(t, name, method.Type().In(i), filled)
			}
			called++
			for _, r := range callRecovering(t, name, method, args) {
				if err, ok := r.Interface().(error); ok && err != nil &&
					strings.Contains(strings.ToLower(err.Error()), "readonly database") {
					t.Errorf("%s schreibt, obwohl es als lesend eingeordnet ist: %v", name, err)
				}
			}
		}
	}
	if called < 200 {
		t.Fatalf("nur %d Aufrufe", called)
	}

	// Dieselben Wege mit dem vorhandenen Beleg und der Buchung, nicht nur mit
	// Kennung 0.
	receiptID := request.ReceiptID
	for name, call := range map[string]func() error{
		"GetReceiptPreview":    func() error { _, err := b.GetReceiptPreview(receiptID); return err },
		"GetReceiptFindings":   func() error { _, err := b.GetReceiptFindings(receiptID); return err },
		"GetEntriesForReceipt": func() error { _, err := b.GetEntriesForReceipt(receiptID); return err },
		"GetAuditTrail":        func() error { _, err := b.GetAuditTrail(receiptID); return err },
		"ProposeFromEInvoice":  func() error { _, err := b.ProposeFromEInvoice(receiptID); return err },
		"GetPaymentAllocations": func() error {
			_, err := b.GetPaymentAllocations(entry.ID)
			return err
		},
		"VerifyIntegrity": func() error {
			result, err := b.VerifyIntegrity()
			if err == nil && !result.IsValid {
				return fmt.Errorf("Kette gebrochen: %s", result.Message)
			}
			return err
		},
	} {
		if err := call(); err != nil && strings.Contains(strings.ToLower(err.Error()), "readonly database") {
			t.Errorf("%s schreibt, obwohl es als lesend eingeordnet ist: %v", name, err)
		} else if name == "VerifyIntegrity" && err != nil {
			t.Errorf("VerifyIntegrity: %v", err)
		}
	}
}

func callRecovering(t *testing.T, name string, method reflect.Value, args []reflect.Value) (out []reflect.Value) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s bricht mit Nullwerten ab: %v", name, fmt.Sprint(r))
			out = nil
		}
	}()
	return method.Call(args)
}

// argument baut einen Aufrufwert: Nullwert oder ein Wert aus dem Bestand.
// Exporte und Ausgaben bekommen einen Pfad im Testordner, damit nichts im
// Paketordner landet.
func argument(t *testing.T, method string, in reflect.Type, filled bool) reflect.Value {
	if in.Kind() == reflect.String && (strings.HasPrefix(method, "Export") || strings.HasPrefix(method, "Save")) {
		return reflect.ValueOf(filepath.Join(t.TempDir(), "ausgabe")).Convert(in)
	}
	v := reflect.New(in).Elem()
	if !filled {
		return v
	}
	switch in.Kind() {
	case reflect.Int, reflect.Int64:
		v.SetInt(2026)
	case reflect.Uint, reflect.Uint64:
		v.SetUint(1)
	case reflect.String:
		v.SetString("2026-12-31")
	}
	return v
}
