package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/buchfink/buchfink/internal/actor"
	"github.com/buchfink/buchfink/internal/core"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/mcpserver/oauth"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Jedes Werkzeug läuft über denselben Rahmen: Anmeldung lesen, Mandant mit
// Lease holen, Zulassung prüfen, Erfassungsweg agent:<kurzname> setzen und
// über Tenant.Read oder Tenant.Write in den Kern. Schreibende Werkzeuge
// verlangen mandant_id und nehmen einen Idempotenzschlüssel an.

// writeTimeout begrenzt das Warten eines Agenten auf die Schreibsperre samt
// Vorgang. Danach antwortet das Werkzeug mit „beschaeftigt".
const writeTimeout = 30 * time.Second

// Tier ist die Freigabestufe eines Werkzeugs.
type Tier string

const (
	// TierRead ändert nichts an den Büchern.
	TierRead Tier = "lesen"
	// TierDirect schreibt ohne Rückfrage; Buchfink prüft die Regeln für
	// Routine (internal/core/policy) und antwortet sonst mit freigabe_noetig.
	TierDirect Tier = "direkt"
	// TierConfirm schreibt erst, nachdem der Agent im Chat das Ja des
	// Menschen eingeholt hat; destructiveHint ist gesetzt.
	TierConfirm Tier = "rueckfrage"
)

// Call ist der Zusammenhang eines Werkzeugaufrufs.
type Call struct {
	Tool     string
	ClientID string
	App      string
	Channel  actor.Channel
	// Scopes sind die Rechte im geöffneten Mandanten: was die Anmeldung trägt
	// und die Zulassung für diesen Mandanten erlaubt. Ohne zugelassenen
	// Mandanten leer.
	Scopes []string
	// Tenant ist der geöffnete Mandant, solange der Aufruf läuft; nil, wenn
	// keiner geöffnet ist.
	Tenant *core.Tenant
	// Granted: die App ist für diesen Mandanten zugelassen.
	Granted bool
	// Host ist der Kern hinter dem Mandanten.
	Host *core.Host
}

// Can meldet, ob die App im geöffneten Mandanten einen Scope hat.
func (c *Call) Can(scope string) bool { return slices.Contains(c.Scopes, scope) }

// Spec beschreibt ein Werkzeug.
type Spec struct {
	Name        string
	Title       string
	Description string
	Tier        Tier
	// Idempotent: derselbe Aufruf ein zweites Mal ändert nichts mehr.
	Idempotent bool
	// ExaminerExempt: das Werkzeug ändert die Bücher nicht (Export) und läuft
	// deshalb auch im Prüfermodus.
	ExaminerExempt bool

	output *jsonschema.Schema
}

// OutputSchema ist das Schema der Ausgabe eines Werkzeugs.
func (tb *Toolbox) OutputSchema(name string) *jsonschema.Schema {
	for _, s := range tb.specs {
		if s.Name == name {
			return s.output
		}
	}
	return nil
}

// TenantArgs gehören in die Eingabe jedes schreibenden Werkzeugs.
type TenantArgs struct {
	MandantID string `json:"mandant_id" jsonschema:"id aus status; ist ein anderer Mandant geöffnet, wird nichts geschrieben"`
}

func (a TenantArgs) tenantID() string { return a.MandantID }

type tenantInput interface{ tenantID() string }

// WriteArgs gehören in die Eingabe jedes schreibenden Werkzeugs, das in einem
// Vorgang schreibt.
type WriteArgs struct {
	MandantID            string `json:"mandant_id" jsonschema:"id aus status; ist ein anderer Mandant geöffnet, wird nichts geschrieben"`
	Idempotenzschluessel string `json:"idempotenz_schluessel,omitempty" jsonschema:"8 bis 100 Zeichen (A-Z a-z 0-9 - _ . :); gleicher Schlüssel und gleiche Eingabe liefern das erste Ergebnis"`
}

func (a WriteArgs) writeArgs() WriteArgs { return a }
func (a WriteArgs) tenantID() string     { return a.MandantID }
func (a WriteArgs) keyRequired() bool    { return false }

// KeyedWriteArgs sind WriteArgs mit Pflichtschlüssel. Sie gehören in die
// Eingabe eines Werkzeugs, dessen Vorgang keinen natürlichen Schlüssel hat:
// ohne ihn buchte ein nach einem Abbruch wiederholter Aufruf ein zweites Mal.
type KeyedWriteArgs struct {
	MandantID            string `json:"mandant_id" jsonschema:"id aus status; ist ein anderer Mandant geöffnet, wird nichts geschrieben"`
	Idempotenzschluessel string `json:"idempotenz_schluessel" jsonschema:"Pflicht, einer je Vorgang: 8 bis 100 Zeichen (A-Z a-z 0-9 - _ . :); gleicher Schlüssel und gleiche Eingabe liefern das erste Ergebnis; eine Vorschau verbraucht ihn nicht"`
}

func (a KeyedWriteArgs) writeArgs() WriteArgs {
	return WriteArgs{MandantID: a.MandantID, Idempotenzschluessel: a.Idempotenzschluessel}
}
func (a KeyedWriteArgs) tenantID() string  { return a.MandantID }
func (a KeyedWriteArgs) keyRequired() bool { return true }

type writeInput interface {
	tenantInput
	writeArgs() WriteArgs
	keyRequired() bool
}

// previewInput ist eine Eingabe mit nur_vorschau. Eine Vorschau schreibt
// nichts und geht deshalb am Idempotenzdienst vorbei: derselbe Schlüssel
// gilt danach für den echten Aufruf.
type previewInput interface{ preview() bool }

// Wiederholung kennzeichnet die Ausgabe eines schreibenden Werkzeugs, die aus
// einem früheren Aufruf mit demselben Schlüssel stammt.
type Wiederholung struct {
	Wiederholt bool `json:"wiederholt,omitempty" jsonschema:"true, wenn dieser Vorgang unter dem Schlüssel schon lief; die Ausgabe ist die des ersten Laufs"`
}

func (w *Wiederholung) markReplayed() { w.Wiederholt = true }

type replayMarker interface{ markReplayed() }

// Referenced ist eine Ausgabe, die auf ein geschriebenes Objekt zeigt
// (z. B. „buchung:812"); sie steht am Idempotenzschlüssel.
type Referenced interface{ Ref() string }

// Toolbox registriert die Werkzeuge am MCP-Server.
type Toolbox struct {
	host  *core.Host
	store *oauth.Store
	srv   *mcp.Server
	specs []Spec
}

// Names sind die registrierten Werkzeuge.
func (tb *Toolbox) Names() []string {
	out := make([]string, 0, len(tb.specs))
	for _, s := range tb.specs {
		out = append(out, s.Name)
	}
	return out
}

// Specs sind die Beschreibungen der registrierten Werkzeuge.
func (tb *Toolbox) Specs() []Spec { return slices.Clone(tb.specs) }

// AddInfo registriert ein lesendes Werkzeug, das auch ohne geöffneten oder
// freigegebenen Mandanten antwortet.
func AddInfo[In, Out any](tb *Toolbox, spec Spec, fn func(ctx context.Context, c *Call, in In) (Out, error)) {
	spec.Tier = TierRead
	add(tb, spec, fn)
}

// AddRead registriert ein lesendes Werkzeug im geöffneten Mandanten.
func AddRead[In, Out any](tb *Toolbox, spec Spec, fn func(ctx context.Context, c *Call, s *core.Services, in In) (Out, error)) {
	spec.Tier = TierRead
	add(tb, spec, func(ctx context.Context, c *Call, in In) (Out, error) {
		var out Out
		if err := c.ensureTenant(); err != nil {
			return out, err
		}
		if !c.Can(oauth.ScopeRead) {
			return out, errScope(oauth.ScopeRead)
		}
		err := c.Tenant.Read(ctx, func(ctx context.Context, s *core.Services) error {
			var err error
			out, err = fn(ctx, c, s, in)
			return err
		})
		return out, err
	})
}

// AddWrite registriert ein schreibendes Werkzeug. Es läuft als ein Vorgang
// über Tenant.Write und damit unter Schreibsperre, Prüfermodus und einer
// Transaktion; der Idempotenzschlüssel steht in derselben Transaktion.
func AddWrite[In writeInput, Out any](tb *Toolbox, spec Spec, fn func(ctx context.Context, c *Call, s *core.Services, in In) (Out, error)) {
	AddWritePrepared(tb, spec, nil, fn)
}

// AddWritePrepared ist AddWrite mit einem Vorlauf ohne Schreibsperre, etwa
// für einen Kurs aus dem Netz. Der Vorlauf schreibt nicht in die Bücher.
func AddWritePrepared[In writeInput, Out any](
	tb *Toolbox, spec Spec,
	prepare func(ctx context.Context, c *Call, t *core.Tenant, in In) error,
	fn func(ctx context.Context, c *Call, s *core.Services, in In) (Out, error),
) {
	checkWriteTier(spec)
	add(tb, spec, func(ctx context.Context, c *Call, in In) (Out, error) {
		var out Out
		if err := c.ensureWritable(in, spec.ExaminerExempt); err != nil {
			return out, err
		}
		args := in.writeArgs()
		preview := false
		if p, ok := any(in).(previewInput); ok {
			preview = p.preview()
		}
		switch {
		case args.Idempotenzschluessel != "":
			if err := domain.ValidateIdempotencyKey(args.Idempotenzschluessel); err != nil {
				return out, inputError("idempotenz_schluessel", "%v", err)
			}
		case in.keyRequired() && !preview:
			return out, inputError("idempotenz_schluessel", "dieses Werkzeug bucht ohne natürlichen Schlüssel; "+
				"ein eigener Schlüssel je Vorgang ist Pflicht, damit eine Wiederholung nicht doppelt bucht")
		}
		ctx, cancel := context.WithTimeout(ctx, writeTimeout)
		defer cancel()
		if prepare != nil {
			if err := prepare(ctx, c, c.Tenant, in); err != nil {
				return out, err
			}
		}
		err := c.Tenant.Write(ctx, func(ctx context.Context, s *core.Services) error {
			if args.Idempotenzschluessel == "" || preview {
				var err error
				out, err = fn(ctx, c, s, in)
				return err
			}
			request, err := withoutKey(in)
			if err != nil {
				return err
			}
			outcome, err := s.Idempotency.Run(ctx, service.IdempotentCall{
				ClientID: c.ClientID, Key: args.Idempotenzschluessel, Operation: spec.Name, Request: request,
			}, func(ctx context.Context) (string, any, error) {
				result, err := fn(ctx, c, s, in)
				ref := ""
				if r, ok := any(result).(Referenced); ok {
					ref = r.Ref()
				}
				return ref, result, err
			})
			if err != nil {
				return err
			}
			if err := json.Unmarshal(outcome.Response, &out); err != nil {
				return fmt.Errorf("das gespeicherte Ergebnis ließ sich nicht lesen: %w", err)
			}
			if outcome.Replayed {
				if m, ok := any(&out).(replayMarker); ok {
					m.markReplayed()
				}
			}
			return nil
		})
		return out, err
	})
}

// AddFlow registriert ein schreibendes Werkzeug aus mehreren Schritten, das
// selbst über den Mandanten schreibt: Dateien außerhalb der Schreibsperre,
// Vermerk und Buchung in eigenen Vorgängen. Es prüft Mandant, Zulassung und
// Recht wie AddWrite; einen Idempotenzschlüssel nimmt es nicht, weil seine
// Schritte schon von sich aus nichts doppelt schreiben.
func AddFlow[In tenantInput, Out any](tb *Toolbox, spec Spec, fn func(ctx context.Context, c *Call, t *core.Tenant, in In) (Out, error)) {
	checkWriteTier(spec)
	add(tb, spec, func(ctx context.Context, c *Call, in In) (Out, error) {
		var out Out
		if err := c.ensureWritable(in, spec.ExaminerExempt); err != nil {
			return out, err
		}
		ctx, cancel := context.WithTimeout(ctx, writeTimeout)
		defer cancel()
		return fn(ctx, c, c.Tenant, in)
	})
}

func checkWriteTier(spec Spec) {
	if spec.Tier != TierDirect && spec.Tier != TierConfirm {
		panic(fmt.Sprintf("Werkzeug %s: ein schreibendes Werkzeug ist direkt oder rueckfrage, nicht %q", spec.Name, spec.Tier))
	}
}

// withoutKey ist die Eingabe ohne den Schlüssel: der Schlüssel benennt den
// Vorgang, er gehört nicht zu dem, was verglichen wird.
func withoutKey(in any) (map[string]any, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	delete(m, "idempotenz_schluessel")
	return m, nil
}

func (c *Call) ensureTenant() error {
	if c.Tenant == nil {
		return errNoTenant()
	}
	if !c.Granted {
		return errNotGranted()
	}
	return nil
}

// ensureWritable prüft vor jeder Eingabeprüfung, ob überhaupt geschrieben
// werden darf; im Prüfermodus erfährt der Agent das sofort.
func (c *Call) ensureWritable(in tenantInput, examinerExempt bool) error {
	if c.Tenant == nil {
		return errNoTenant()
	}
	if id := in.tenantID(); id != c.Tenant.ID() {
		return errSwitched(id, c.Tenant.ID())
	}
	if !c.Granted {
		return errNotGranted()
	}
	if !c.Can(oauth.ScopeWrite) {
		return errScope(oauth.ScopeWrite)
	}
	if ex := c.Tenant.Examiner(); ex.Active && !examinerExempt {
		return &ToolError{Code: CodeExaminer,
			Nachricht:        "Der Prüfermodus ist bis " + ex.Until + " aktiv; Buchfink nimmt keine Änderung an.",
			NaechsterSchritt: "Bis zum Ende des Prüfermodus nur lesen. Beenden lässt er sich nur in der App.", Werkzeug: "nur_app"}
	}
	return nil
}

func add[In, Out any](tb *Toolbox, spec Spec, run func(ctx context.Context, c *Call, in In) (Out, error)) {
	if spec.Tier == "" || spec.Title == "" || spec.Description == "" {
		panic(fmt.Sprintf("Werkzeug %s: Stufe, Titel und Beschreibung sind Pflicht", spec.Name))
	}
	inSchema, err := jsonschema.For[In](schemaOptions())
	if err != nil {
		panic(fmt.Sprintf("Werkzeug %s: Eingabeschema: %v", spec.Name, err))
	}
	outSchema, err := jsonschema.For[Out](outputSchemaOptions())
	if err != nil {
		panic(fmt.Sprintf("Werkzeug %s: Ausgabeschema: %v", spec.Name, err))
	}
	closed := false
	annotations := &mcp.ToolAnnotations{Title: spec.Title, ReadOnlyHint: spec.Tier == TierRead, OpenWorldHint: &closed}
	if spec.Tier == TierRead {
		annotations.IdempotentHint = true
	} else {
		destructive := spec.Tier == TierConfirm
		annotations.DestructiveHint = &destructive
		annotations.IdempotentHint = spec.Idempotent
	}
	// Das Ausgabeschema wird nicht veröffentlicht: der Werkzeugkatalog liegt
	// in jedem Gespräch im Kontext des Agenten, und die Ausgaben machten gut
	// die Hälfte davon aus. Die Tests prüfen jede Antwort gegen das Schema
	// (OutputSchema).
	tool := &mcp.Tool{
		Name: spec.Name, Title: spec.Title, Description: spec.Description,
		Annotations: annotations, InputSchema: inSchema,
	}
	spec.output = outSchema
	mcp.AddTool(tb.srv, tool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		c, release, err := tb.newCall(ctx, req, spec.Name)
		if err != nil {
			return errorResult(toolError(err)), nil, nil
		}
		defer release()
		out, err := run(actor.WithChannel(ctx, c.Channel), c, in)
		if err != nil {
			te := toolError(err)
			if _, keyed := any(in).(writeInput); te.Code == CodeBusy && !keyed {
				te.NaechsterSchritt = "In einigen Sekunden erneut aufrufen; das Werkzeug schreibt nichts doppelt."
			}
			return errorResult(te), nil, nil
		}
		body, err := json.Marshal(out)
		if err != nil {
			return errorResult(&ToolError{Code: CodeInternal, Nachricht: "die Antwort ließ sich nicht aufbereiten"}), nil, nil
		}
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: string(body)}},
			StructuredContent: json.RawMessage(body),
		}, nil, nil
	})
	tb.specs = append(tb.specs, spec)
}

// newCall liest Anmeldung und offenen Mandanten. Die Rechte gelten je
// Mandant: ein Token hängt an der App, nicht am Mandanten, für den sie
// zugelassen wurde. release gibt die Lease zurück.
func (tb *Toolbox) newCall(ctx context.Context, req *mcp.CallToolRequest, tool string) (*Call, func(), error) {
	info := req.Extra.TokenInfo
	if info == nil {
		return nil, nil, &ToolError{Code: CodeMissingScope, Nachricht: "der Aufruf trägt keine Anmeldung"}
	}
	slug, _ := info.Extra["slug"].(string)
	channel, err := actor.Agent(slug)
	if err != nil {
		return nil, nil, &ToolError{Code: CodeInternal, Nachricht: "die Anmeldung nennt keinen gültigen Kurznamen: " + err.Error()}
	}
	name, _ := info.Extra["name"].(string)
	c := &Call{Tool: tool, ClientID: info.UserID, App: name, Channel: channel, Host: tb.host}
	t, release, err := tb.host.Acquire(ctx)
	switch {
	case errors.Is(err, core.ErrNoTenant), errors.Is(err, core.ErrTenantClosing):
		return c, func() {}, nil
	case err != nil:
		return nil, nil, err
	}
	c.Tenant = t
	grant, err := tb.store.ActiveGrant(ctx, c.ClientID, t.ID())
	if err != nil {
		release()
		return nil, nil, err
	}
	if grant != nil {
		c.Granted = true
		allowed := strings.Fields(grant.Scopes)
		for _, sc := range info.Scopes {
			if slices.Contains(allowed, sc) {
				c.Scopes = append(c.Scopes, sc)
			}
		}
	}
	return c, release, nil
}

// Seite ist die Eingabe eines seitenweise antwortenden Werkzeugs.
type Seite struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"1 bis 100, Vorgabe 25"`
	Cursor string `json:"cursor,omitempty" jsonschema:"naechste_seite der vorigen Antwort"`
}

// Seitenende gehört in die Ausgabe eines seitenweise antwortenden Werkzeugs.
type Seitenende struct {
	NaechsteSeite *string `json:"naechste_seite" jsonschema:"Cursor für die nächste Seite; null, wenn dies die letzte ist"`
	Gesamt        int     `json:"gesamt" jsonschema:"Zahl aller Treffer"`
}

const (
	defaultPageSize = 25
	maxPageSize     = 100
)

// Paginate schneidet eine Seite aus einer Liste. Der Cursor ist
// undurchsichtig; ein fremder oder verbogener Cursor ist ein Eingabefehler.
// Die Seite ist nie nil.
func Paginate[T any](items []T, p Seite) ([]T, Seitenende, error) {
	limit := p.Limit
	if limit == 0 {
		limit = defaultPageSize
	}
	if limit < 1 || limit > maxPageSize {
		return nil, Seitenende{}, &ToolError{Code: CodeInvalidInput,
			Nachricht: fmt.Sprintf("limit muss zwischen 1 und %d liegen, nicht %d", maxPageSize, p.Limit)}
	}
	offset := 0
	if p.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(p.Cursor)
		n, convErr := strconv.Atoi(strings.TrimPrefix(string(raw), "ab:"))
		if err != nil || convErr != nil || !strings.HasPrefix(string(raw), "ab:") || n < 0 || n > len(items) {
			return nil, Seitenende{}, &ToolError{Code: CodeInvalidInput, Nachricht: "der Cursor ist ungültig",
				NaechsterSchritt: "Ohne cursor mit der ersten Seite neu beginnen."}
		}
		offset = n
	}
	end := min(offset+limit, len(items))
	page := append(make([]T, 0, end-offset), items[offset:end]...)
	if end < len(items) {
		next := base64.RawURLEncoding.EncodeToString([]byte("ab:" + strconv.Itoa(end)))
		return page, Seitenende{NaechsteSeite: &next, Gesamt: len(items)}, nil
	}
	return page, Seitenende{Gesamt: len(items)}, nil
}
