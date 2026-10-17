package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Die Brücke merkt sich je Buchfink-Adresse die Registrierung, die Tokens und
// die zuletzt gesehene Werkzeugliste. Kein Schlüsselbund: die Bücher selbst
// liegen unverschlüsselt auf demselben Rechner, und das Refresh-Token ist an
// diese App, Loopback-Rücksprünge und den Server auf 127.0.0.1 gebunden.

// state ist der Inhalt der Anmeldedatei.
type state struct {
	Endpoint      string     `json:"endpoint"`
	Issuer        string     `json:"issuer,omitempty"`
	ClientID      string     `json:"client_id,omitempty"`
	TokenEndpoint string     `json:"token_endpoint,omitempty"`
	Token         *tokenData `json:"token,omitempty"`
	Updated       time.Time  `json:"aktualisiert"`
}

type tokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Expiry       time.Time `json:"expiry"`
	Scope        string    `json:"scope,omitempty"`
}

// stateStore liest und schreibt die Anmeldedatei. Jede Änderung geschieht
// unter einer Dateisperre: zwei Brücken-Prozesse dürfen nicht dasselbe
// Refresh-Token einlösen, sonst widerruft der Server die ganze Kette.
type stateStore struct {
	dir      string
	path     string
	lockPath string
	tools    string
}

func newStateStore(dir, endpoint string) (*stateStore, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("der Ordner %s ließ sich nicht anlegen: %w", dir, err)
	}
	sum := sha256.Sum256([]byte(endpoint))
	name := hex.EncodeToString(sum[:])[:12]
	return &stateStore{
		dir:      dir,
		path:     filepath.Join(dir, name+".json"),
		lockPath: filepath.Join(dir, name+".lock"),
		tools:    filepath.Join(dir, name+"-werkzeuge.json"),
	}, nil
}

func (s *stateStore) load(endpoint string) (*state, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &state{Endpoint: endpoint}, nil
	}
	if err != nil {
		return nil, err
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		// Eine kaputte Datei heißt nur: neu anmelden.
		return &state{Endpoint: endpoint}, nil
	}
	if st.Endpoint != endpoint {
		return &state{Endpoint: endpoint}, nil
	}
	return &st, nil
}

func (s *stateStore) save(st *state) error {
	st.Updated = time.Now().UTC()
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, data)
}

// lock sperrt die Anmeldedatei für diesen Prozess; unlock gibt sie frei.
func (s *stateStore) lock() (unlock func(), err error) {
	f, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = unlockFile(f)
		_ = f.Close()
	}, nil
}

// writeFileAtomic schreibt über eine Temp-Datei und Umbenennen, mit 0600.
func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
