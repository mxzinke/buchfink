package core

import (
	"context"
	"errors"
	"strings"

	"github.com/buchfink/buchfink/internal/currency"
	"github.com/buchfink/buchfink/internal/domain"
	"github.com/buchfink/buchfink/internal/service"
	"github.com/buchfink/buchfink/internal/timestamp"
	"github.com/buchfink/buchfink/internal/vatid"
)

// ErrNetworkInWrite: ein Netzaufruf aus einem laufenden Schreibvorgang. Er
// hielte die Schreibsperre für die Dauer seines Zeitlimits (bis zu 20 s), und
// jeder andere Schreiber, auch die Oberfläche, wartete mit.
var ErrNetworkInWrite = errors.New(
	"ein Netzaufruf innerhalb eines Schreibvorgangs ist nicht zulässig; der Wert wird vor dem Schreiben geholt")

// Network sind die Wege ins Netz: Zeitstempelstelle, Referenzkurse der EZB und
// die Bestätigungsanfrage beim Bundeszentralamt. Ein leeres Feld nimmt den
// echten Dienst; Tests setzen eigene.
//
// Jeder Weg weist einen Aufruf aus einem laufenden Schreibvorgang ab. Das Netz
// wird vorher gefragt (PrefetchRate, CheckVatID, PrepareInvoice, Festschreibung
// in drei Schritten); im Schreibvorgang zählt, was abgelegt ist.
type Network struct {
	Timestamp func(ctx context.Context, tsaURL, sha256Hex string) (*timestamp.Result, error)
	Rates     domain.CurrencyFetcher
	VatID     func() service.VatIDClient
}

// ErrRatesInExaminerMode: im Prüfermodus holt der Kern keinen Kurs. Ein
// geholter Kurs würde abgelegt und änderte so den Datenbestand.
var ErrRatesInExaminerMode = errors.New(
	"solange der Prüfermodus läuft, holt Buchfink keinen Kurs nach — ein geholter Kurs wäre eine Änderung am Datenbestand")

// guarded ergänzt die fehlenden Wege und legt den Wächter davor. Kurse holt
// er nicht, solange examinerActive den Prüfermodus meldet; das gilt auch für
// Lesevorgänge, die einen geholten Kurs sonst ablegen.
func (n Network) guarded(examinerActive func() bool) Network {
	if n.Timestamp == nil {
		n.Timestamp = timestamp.RequestToken
	}
	if n.Rates == nil {
		n.Rates = currency.New(currency.DefaultEndpoint)
	}
	if n.VatID == nil {
		n.VatID = func() service.VatIDClient { return vatid.New(vatid.DefaultEndpoint) }
	}
	stamp, rates, vatIDs := n.Timestamp, n.Rates, n.VatID
	return Network{
		Timestamp: func(ctx context.Context, tsaURL, hash string) (*timestamp.Result, error) {
			if InWrite(ctx) {
				return nil, ErrNetworkInWrite
			}
			return stamp(ctx, tsaURL, hash)
		},
		Rates: guardedRates{inner: rates, examinerActive: examinerActive},
		VatID: func() service.VatIDClient { return guardedVatID{vatIDs()} },
	}
}

type guardedRates struct {
	inner          domain.CurrencyFetcher
	examinerActive func() bool
}

func (g guardedRates) RateAt(ctx context.Context, currency, date string) (int64, string, string, error) {
	if InWrite(ctx) {
		return 0, "", "", ErrNetworkInWrite
	}
	if g.examinerActive != nil && g.examinerActive() {
		return 0, "", "", ErrRatesInExaminerMode
	}
	return g.inner.RateAt(ctx, currency, date)
}

type guardedVatID struct{ inner service.VatIDClient }

func (g guardedVatID) Check(ctx context.Context, req vatid.Request) (*vatid.Result, error) {
	if InWrite(ctx) {
		return nil, ErrNetworkInWrite
	}
	return g.inner.Check(ctx, req)
}

func (g guardedVatID) Endpoint() string { return g.inner.Endpoint() }

// PrefetchRate holt den Referenzkurs eines Tages ohne Schreibsperre und legt
// ihn in der Kurshistorie ab, damit ein folgender Schreibvorgang ihn dort
// findet. Euro braucht keinen. Im Prüfermodus wird nichts geholt: der
// Schreibvorgang danach wird ohnehin abgewiesen.
func (t *Tenant) PrefetchRate(ctx context.Context, code, date string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || code == "EUR" || date == "" || t.Examiner().Active {
		return nil
	}
	return t.Read(ctx, func(ctx context.Context, s *Services) error {
		_, err := s.Currency.RateAt(ctx, code, date)
		return err
	})
}

// PrefetchValuationRates holt die Stichtagskurse der Fremdwährungsbewertung
// eines Geschäftsjahres, bevor sie gebucht wird.
func (t *Tenant) PrefetchValuationRates(ctx context.Context, year int) error {
	if t.Examiner().Active {
		return nil
	}
	return t.Read(ctx, func(ctx context.Context, s *Services) error {
		_, err := s.Currency.PreviewCurrencyValuation(ctx, year)
		return err
	})
}

// CheckVatID fragt die USt-IdNr. eines Geschäftspartners beim
// Bundeszentralamt ab: vorbereitet im Lesen, gefragt ohne Sperre, abgelegt in
// einem kurzen Schreibvorgang. Auch eine ausgebliebene Antwort steht danach im
// Protokoll.
func (t *Tenant) CheckVatID(ctx context.Context, contactID uint) (*domain.VatIDCheck, error) {
	if err := examinerError(t.Examiner()); err != nil {
		return nil, err
	}
	var query *service.VatIDQuery
	if err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		var err error
		query, err = s.VatIDs.PrepareCheck(ctx, contactID)
		return err
	}); err != nil {
		return nil, err
	}
	result, runErr := query.Run(ctx)
	var check *domain.VatIDCheck
	err := t.Write(ctx, func(ctx context.Context, s *Services) error {
		var err error
		check, err = s.VatIDs.RecordCheck(ctx, query, result, runErr)
		return err
	})
	return check, err
}

// PrepareInvoice holt vor dem Ausstellen ohne Schreibsperre, was eine Rechnung
// aus dem Netz braucht: den Kurs einer Fremdwährungsrechnung und die
// Bestätigung der USt-IdNr. bei einer Leistung in einen anderen Mitgliedstaat.
//
// Bleibt die Bestätigung aus, entscheidet das Ausstellen: mit festgehaltenem
// Grund geht die Rechnung hinaus, ohne nicht. Fehlt der Kurs, kann nicht
// gebucht werden, und der Fehler kommt hier zurück.
func (t *Tenant) PrepareInvoice(ctx context.Context, inv *domain.Invoice) error {
	if inv == nil || t.Examiner().Active {
		return nil
	}
	if err := t.PrefetchRate(ctx, inv.Currency, inv.Date); err != nil {
		return err
	}
	var contactID uint
	if err := t.Read(ctx, func(ctx context.Context, s *Services) error {
		var err error
		contactID, err = s.Invoices.VatIDToConfirm(ctx, inv)
		return err
	}); err != nil || contactID == 0 {
		return err
	}
	if _, err := t.CheckVatID(ctx, contactID); err != nil {
		t.logger.Info("USt-IdNr. vor dem Ausstellen nicht bestätigt", "kontakt", contactID, "fehler", err)
	}
	return nil
}

// ErrOffline ist die Antwort von OfflineNetwork.
var ErrOffline = errors.New("kein Netz")

// OfflineNetwork ist ein Netz, das nie antwortet. Für Tests und Läufe, die
// nichts nach draußen schicken dürfen.
func OfflineNetwork() Network {
	return Network{
		Timestamp: func(context.Context, string, string) (*timestamp.Result, error) { return nil, ErrOffline },
		Rates:     offlineRates{},
		VatID:     func() service.VatIDClient { return offlineVatID{} },
	}
}

type offlineRates struct{}

func (offlineRates) RateAt(context.Context, string, string) (int64, string, string, error) {
	return 0, "", "", ErrOffline
}

type offlineVatID struct{}

func (offlineVatID) Check(context.Context, vatid.Request) (*vatid.Result, error) {
	return nil, ErrOffline
}
func (offlineVatID) Endpoint() string { return "offline" }
