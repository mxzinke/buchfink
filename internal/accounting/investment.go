package accounting

import (
	"errors"
	"fmt"
	"time"

	"github.com/buchfink/buchfink/internal/domain"
)

// Investmentanteile nach dem Investmentsteuergesetz.
//
// Ein ETF steht in der Bilanz wie jedes andere Wertpapier des Anlagevermögens.
// Steuerlich ist er es nicht: das InvStG legt zwei Rechnungen daneben, die in
// keiner Buchung auftauchen und deshalb sonst verloren gehen.
//
//   - Die **Teilfreistellung** (§ 20 InvStG) stellt einen Teil der Erträge und
//     des Veräußerungsgewinns steuerfrei. Wie groß der Teil ist, hängt vom Fonds
//     *und* vom Anleger ab.
//   - Die **Vorabpauschale** (§ 18 InvStG) besteuert einen Mindestertrag, auch
//     wenn der Fonds thesauriert und kein Geld fließt. Weil sie über die Jahre
//     schon versteuert wurde, muss sie beim Abgang wieder abgezogen werden.
//
// Beides sind außerbilanzielle Korrekturen. Buchfink rechnet und dokumentiert
// sie; gebucht wird nichts, weil handelsrechtlich nichts geschieht.

// FundClass is the Fondsart, an der § 20 InvStG die Teilfreistellung festmacht.
type FundClass string

const (
	// FundNone marks a Wertpapier that is no Investmentanteil at all — eine
	// Einzelaktie, eine Anleihe. Für sie gibt es keine Teilfreistellung.
	FundNone FundClass = ""
	// FundEquity ist der Aktienfonds: mindestens 51 % Kapitalbeteiligungen
	// (§ 2 Abs. 6 InvStG).
	FundEquity FundClass = "equity"
	// FundMixed ist der Mischfonds: mindestens 25 % Kapitalbeteiligungen
	// (§ 2 Abs. 7 InvStG). Für ihn gilt die Hälfte des Aktiensatzes.
	FundMixed FundClass = "mixed"
	// FundRealEstate ist der Immobilienfonds (§ 2 Abs. 9 Satz 1 InvStG).
	FundRealEstate FundClass = "real_estate"
	// FundForeignRealEstate ist der Auslands-Immobilienfonds
	// (§ 2 Abs. 9 Satz 2 InvStG).
	FundForeignRealEstate FundClass = "foreign_real_estate"
	// FundOther ist ein Investmentfonds, der keine der Quoten erreicht. Er ist
	// ein Investmentanteil — nur ohne Teilfreistellung.
	FundOther FundClass = "other"
)

// Label renders the fund class for the UI.
func (c FundClass) Label() string {
	switch c {
	case FundEquity:
		return "Aktienfonds (mindestens 51 % Kapitalbeteiligungen)"
	case FundMixed:
		return "Mischfonds (mindestens 25 % Kapitalbeteiligungen)"
	case FundRealEstate:
		return "Immobilienfonds"
	case FundForeignRealEstate:
		return "Auslands-Immobilienfonds"
	case FundOther:
		return "Investmentfonds ohne Teilfreistellung"
	default:
		return "Kein Investmentanteil"
	}
}

// IsFund reports whether the asset is an Investmentanteil at all.
func (c FundClass) IsFund() bool { return c != FundNone }

// Valid reports whether the class is one of the known ones.
func (c FundClass) Valid() bool {
	switch c {
	case FundNone, FundEquity, FundMixed, FundRealEstate, FundForeignRealEstate, FundOther:
		return true
	default:
		return false
	}
}

// AllFundClasses returns the catalog in the order the mask offers it.
func AllFundClasses() []FundClass {
	return []FundClass{
		FundNone, FundEquity, FundMixed, FundRealEstate, FundForeignRealEstate, FundOther,
	}
}

// PartialExemption is the Teilfreistellung of one fund for one investor.
type PartialExemption struct {
	// Permille is the exempt share in Promille — 800 sind 80 %.
	Permille int `json:"permille"`
	// Determined is false where the rate cannot be stated with one number.
	Determined bool `json:"determined"`
	// Source names the provision, Explanation says why this rate and not another.
	Source      string `json:"source"`
	Explanation string `json:"explanation"`
}

// PartialExemptionFor applies § 20 InvStG to a fund and an investor.
//
// Zwei Dinge entscheiden, und beide müssen stimmen. Der Fonds bestimmt, *welche*
// Teilfreistellung greift — die Aktien- oder die Immobilienteilfreistellung, und
// die eine schließt die andere aus (§ 20 Abs. 3 Satz 3 InvStG). Der Anleger
// bestimmt bei der Aktienteilfreistellung ihre *Höhe*: 30 % nach Satz 1, 60 %
// für die natürliche Person im Betriebsvermögen nach Satz 2, 80 % für den
// Anleger, der dem KStG unterliegt, nach Satz 3 — und wieder 30 %, wo die Sätze
// 4 und 5 die Erhöhung zurücknehmen.
//
// Die Immobilienteilfreistellung kennt diese Staffelung nicht: 60 % bzw. 80 %
// gelten für jeden Anleger.
func PartialExemptionFor(fund FundClass, investor domain.InvestorType) (PartialExemption, error) {
	if !fund.Valid() {
		return PartialExemption{}, fmt.Errorf("unbekannte Fondsart %q", fund)
	}
	switch fund {
	case FundNone:
		return PartialExemption{}, fmt.Errorf(
			"eine Teilfreistellung gibt es nur für Investmentanteile (§ 20 InvStG)")
	case FundOther:
		return PartialExemption{
			Permille: 0, Determined: true, Source: "§ 20 InvStG",
			Explanation: "Der Fonds erreicht weder die Aktienfonds- noch die Mischfonds- oder " +
				"Immobilienfondsquote. Damit bleibt es bei der vollen Besteuerung.",
		}, nil
	case FundRealEstate:
		return PartialExemption{
			Permille: 600, Determined: true, Source: "§ 20 Abs. 3 Satz 1 InvStG",
			Explanation: "Bei Immobilienfonds sind 60 % der Erträge steuerfrei. Der Satz hängt " +
				"nicht vom Anleger ab, und er schließt die Aktienteilfreistellung aus " +
				"(§ 20 Abs. 3 Satz 3 InvStG).",
		}, nil
	case FundForeignRealEstate:
		return PartialExemption{
			Permille: 800, Determined: true, Source: "§ 20 Abs. 3 Satz 2 InvStG",
			Explanation: "Bei Auslands-Immobilienfonds sind 80 % der Erträge steuerfrei. Der Satz " +
				"hängt nicht vom Anleger ab, und er schließt die Aktienteilfreistellung aus " +
				"(§ 20 Abs. 3 Satz 3 InvStG).",
		}, nil
	}

	// Aktien- und Mischfonds: hier entscheidet der Anleger über die Höhe.
	base, source, why, err := equityExemption(investor)
	if err != nil {
		return PartialExemption{}, err
	}
	if fund == FundMixed {
		// § 20 Abs. 2 InvStG: die Hälfte des für Aktienfonds geltenden Satzes.
		return PartialExemption{
			Permille: base / 2, Determined: true, Source: source + " i. V. m. § 20 Abs. 2 InvStG",
			Explanation: fmt.Sprintf(
				"Bei Mischfonds ist die Hälfte der Aktienteilfreistellung anzusetzen (§ 20 Abs. 2 "+
					"InvStG). %s Für den Aktienfonds wären es %s.", why, permilleText(base)),
		}, nil
	}
	return PartialExemption{
		Permille: base, Determined: true, Source: source, Explanation: why,
	}, nil
}

// equityExemption is the Aktienteilfreistellung for one investor.
func equityExemption(investor domain.InvestorType) (int, string, string, error) {
	switch investor {
	case domain.InvestorCorporate:
		return 800, "§ 20 Abs. 1 Satz 3 InvStG",
			"Der Anleger unterliegt dem Körperschaftsteuergesetz: die Aktienteilfreistellung " +
				"beträgt 80 %.", nil
	case domain.InvestorIndividualBusiness:
		return 600, "§ 20 Abs. 1 Satz 2 InvStG",
			"Eine natürliche Person hält die Anteile im Betriebsvermögen: die " +
				"Aktienteilfreistellung beträgt 60 %.", nil
	case domain.InvestorBasic:
		return 300, "§ 20 Abs. 1 Satz 1 InvStG",
			"Es bleibt beim Grundsatz von 30 %. Das ist der Satz für Anteile im " +
				"Privatvermögen — und der Satz, auf den § 20 Abs. 1 Sätze 4 und 5 InvStG " +
				"zurückfallen lassen, etwa bei Lebens- und Krankenversicherungsunternehmen, bei " +
				"Kreditinstituten mit Handelsbestand und bei Pensionsfonds.", nil
	case domain.InvestorMixed:
		return 0, "§ 20 Abs. 3a InvStG", "", errors.New(
			"bei einer Personengesellschaft bestimmt sich die Teilfreistellung nach dem einzelnen " +
				"Gesellschafter (§ 20 Abs. 3a InvStG). Ein Satz für die Gesellschaft als Ganzes " +
				"gibt es nicht; Buchfink weist den Ertrag deshalb ungekürzt aus und überlässt die " +
				"Aufteilung der Feststellungserklärung")
	default:
		return 0, "", "", errors.New(
			"für die Teilfreistellung fehlt die Anlegerstellung. § 20 Abs. 1 InvStG staffelt sie: " +
				"30 % im Grundsatz, 60 % für eine natürliche Person mit Anteilen im " +
				"Betriebsvermögen, 80 % für einen Anleger, der dem Körperschaftsteuergesetz " +
				"unterliegt. Aus der Rechtsform allein folgt das nicht — trage sie in den " +
				"Stammdaten ein")
	}
}

// basisPointText renders a Basiszins: 253 Basispunkte sind „2,53 %".
func basisPointText(points int) string {
	return fmt.Sprintf("%d,%02d %%", points/100, points%100)
}

func permilleText(permille int) string {
	whole, frac := permille/10, permille%10
	if frac == 0 {
		return fmt.Sprintf("%d %%", whole)
	}
	return fmt.Sprintf("%d,%d %%", whole, frac)
}

// VorabpauschaleInput is what § 18 InvStG needs to compute one year.
type VorabpauschaleInput struct {
	// Year is the Kalenderjahr the Vorabpauschale is computed for. § 18 InvStG
	// rechnet nach Kalenderjahren, auch bei einem abweichenden Wirtschaftsjahr:
	// der Basiszins wird auf den ersten Börsentag des Jahres bestimmt, und der
	// Zufluss liegt am ersten Werktag des folgenden Kalenderjahres.
	Year int
	// OpeningPrice ist der Rücknahmepreis zu Beginn des Kalenderjahres,
	// ClosingPrice der letzte im Kalenderjahr festgesetzte. Wird kein
	// Rücknahmepreis festgesetzt, tritt der Börsen- oder Marktpreis an seine
	// Stelle (§ 18 Abs. 1 Satz 4 InvStG).
	OpeningPrice domain.Cents
	ClosingPrice domain.Cents
	// Distributions sind die Ausschüttungen des Kalenderjahres.
	Distributions domain.Cents
	// BasisPoints ist der Basiszins in Basispunkten: 253 sind 2,53 %.
	//
	// Promille wären hier zu grob — der Zins wird mit zwei Nachkommastellen im
	// Prozent veröffentlicht, und auf einen sechsstelligen Bestand verschiebt
	// die zweite Stelle das Ergebnis um dreistellige Beträge.
	//
	// Er steht nicht im Gesetz: § 18 Abs. 4 InvStG lässt ihn die Bundesbank auf
	// den ersten Börsentag des Jahres errechnen, und das Bundesministerium der
	// Finanzen veröffentlicht ihn im Bundessteuerblatt. Buchfink kann ihn
	// deshalb nicht mitliefern — er wird eingegeben, mit der Fundstelle daneben.
	BasisPoints int
	// AcquisitionMonth ist der Monat des Erwerbs (1 bis 12), wenn die Anteile in
	// diesem Kalenderjahr erworben wurden; sonst null. § 18 Abs. 2 InvStG kürzt
	// die Vorabpauschale dann um ein Zwölftel je vollem Monat davor.
	AcquisitionMonth int
}

// Vorabpauschale is the computed result, with every step kept.
//
// Die Zwischenschritte stehen nicht zur Zierde da: die Vorabpauschale ist der
// Betrag, den jemand versteuert, ohne dass Geld geflossen ist. Wer sie später
// prüft, will sehen, woraus sie entstand — und beim Abgang wird sie wieder
// abgezogen, was ohne die Herkunft nicht nachvollziehbar wäre.
type Vorabpauschale struct {
	Year int `json:"year"`
	// BasisReturn ist der Basisertrag: Rücknahmepreis zu Jahresbeginn mal
	// 70 % des Basiszinses (§ 18 Abs. 1 Satz 2 InvStG).
	BasisReturn domain.Cents `json:"basisReturn"`
	// Growth ist der Mehrbetrag zwischen dem ersten und dem letzten
	// Rücknahmepreis zuzüglich der Ausschüttungen. Auf ihn ist der Basisertrag
	// begrenzt (Satz 3) — ein Fonds, der an Wert verloren hat, trägt keine
	// Vorabpauschale.
	Growth domain.Cents `json:"growth"`
	// Capped sagt, ob diese Grenze gegriffen hat.
	Capped bool `json:"capped"`
	// Distributions mindern die Vorabpauschale: sie ist der Betrag, um den die
	// Ausschüttungen den Basisertrag *unterschreiten* (Satz 1).
	Distributions domain.Cents `json:"distributions"`
	// MonthsCounted ist die Zahl der Zwölftel im Erwerbsjahr (§ 18 Abs. 2).
	MonthsCounted int `json:"monthsCounted"`
	// Amount ist die Vorabpauschale, AccruedOn der Tag, an dem sie als
	// zugeflossen gilt (§ 18 Abs. 3 InvStG).
	Amount    domain.Cents `json:"amount"`
	AccruedOn string       `json:"accruedOn"`
	// Explanation is the whole computation in one paragraph.
	Explanation string `json:"explanation"`
}

// ComputeVorabpauschale applies § 18 InvStG to one Kalenderjahr.
func ComputeVorabpauschale(in VorabpauschaleInput) (Vorabpauschale, error) {
	if in.Year <= 0 {
		return Vorabpauschale{}, fmt.Errorf("ohne Kalenderjahr lässt sich keine Vorabpauschale rechnen")
	}
	if in.BasisPoints <= 0 {
		return Vorabpauschale{}, fmt.Errorf(
			"für %d fehlt der Basiszins. Er steht nicht im Gesetz: die Bundesbank errechnet ihn auf "+
				"den ersten Börsentag des Jahres, das Bundesministerium der Finanzen veröffentlicht "+
				"ihn im Bundessteuerblatt (§ 18 Abs. 4 InvStG). Trage ihn von dort ein", in.Year)
	}
	if in.OpeningPrice < 0 || in.ClosingPrice < 0 || in.Distributions < 0 {
		return Vorabpauschale{}, fmt.Errorf("Rücknahmepreise und Ausschüttungen können nicht negativ sein")
	}
	if in.AcquisitionMonth < 0 || in.AcquisitionMonth > 12 {
		return Vorabpauschale{}, fmt.Errorf("der Erwerbsmonat liegt zwischen 1 und 12")
	}

	out := Vorabpauschale{
		Year:          in.Year,
		Distributions: in.Distributions,
		MonthsCounted: 12,
		AccruedOn:     firstWorkdayOfYear(in.Year + 1),
	}

	// § 18 Abs. 1 Satz 2: Rücknahmepreis zu Beginn des Kalenderjahres mal
	// 70 Prozent des Basiszinses.
	// 70 % des Basiszinses: Basispunkte sind Hundertstel Prozent, also
	// Zehntausendstel — mal 70, geteilt durch 100.
	basis := domain.MulRound(in.OpeningPrice, int64(in.BasisPoints)*70, 10_000*100)

	// Satz 3: begrenzt auf den Wertzuwachs zuzüglich der Ausschüttungen.
	out.Growth = in.ClosingPrice - in.OpeningPrice + in.Distributions
	if out.Growth < 0 {
		out.Growth = 0
	}
	if basis > out.Growth {
		basis = out.Growth
		out.Capped = true
	}
	out.BasisReturn = basis

	// Satz 1: die Vorabpauschale ist der Betrag, um den die Ausschüttungen den
	// Basisertrag unterschreiten. Schüttet der Fonds mehr aus, bleibt nichts.
	amount := basis - in.Distributions
	if amount < 0 {
		amount = 0
	}

	// § 18 Abs. 2: im Jahr des Erwerbs ein Zwölftel weniger für jeden vollen
	// Monat, der dem Erwerbsmonat vorangeht.
	if in.AcquisitionMonth > 0 {
		out.MonthsCounted = 13 - in.AcquisitionMonth
		amount = domain.MulRound(amount, int64(out.MonthsCounted), 12)
	}
	out.Amount = amount

	switch {
	case out.Amount == 0 && in.Distributions >= out.BasisReturn && out.BasisReturn > 0:
		out.Explanation = fmt.Sprintf(
			"Der Basisertrag für %d beträgt %s €; ausgeschüttet wurden %s €. Die Ausschüttungen "+
				"unterschreiten ihn nicht, es entsteht keine Vorabpauschale (§ 18 Abs. 1 Satz 1 InvStG).",
			in.Year, out.BasisReturn, in.Distributions)
	case out.Amount == 0 && out.Growth == 0:
		out.Explanation = fmt.Sprintf(
			"Der Fonds hat %d keinen Wertzuwachs erzielt. Der Basisertrag ist auf den Mehrbetrag "+
				"begrenzt (§ 18 Abs. 1 Satz 3 InvStG) und damit null — eine Vorabpauschale entsteht nicht.",
			in.Year)
	default:
		parts := fmt.Sprintf(
			"Basisertrag %d: %s € × 70 %% von %s = %s €",
			in.Year, in.OpeningPrice, basisPointText(in.BasisPoints), out.BasisReturn)
		if out.Capped {
			parts += fmt.Sprintf(", begrenzt auf den Wertzuwachs von %s € (§ 18 Abs. 1 Satz 3 InvStG)",
				out.Growth)
		}
		if in.Distributions > 0 {
			parts += fmt.Sprintf(", abzüglich Ausschüttungen von %s €", in.Distributions)
		}
		if out.MonthsCounted < 12 {
			parts += fmt.Sprintf(", gekürzt auf %d Zwölftel für das Erwerbsjahr (§ 18 Abs. 2 InvStG)",
				out.MonthsCounted)
		}
		out.Explanation = fmt.Sprintf("%s. Vorabpauschale %s €, zugeflossen am %s "+
			"(§ 18 Abs. 3 InvStG). Handelsrechtlich ist das kein Ertrag — es wird nichts gebucht.",
			parts, out.Amount, out.AccruedOn)
	}
	return out, nil
}

// firstWorkdayOfYear is the day the Vorabpauschale is deemed to accrue on
// (§ 18 Abs. 3 InvStG): der erste Werktag des folgenden Kalenderjahres.
//
// Werktag heißt Montag bis Samstag; gesetzliche Feiertage sind Landesrecht und
// bleiben hier außen vor. Der 1. Januar ist bundeseinheitlich Feiertag, deshalb
// beginnt die Suche am 2. Januar.
func firstWorkdayOfYear(year int) string {
	day := time.Date(year, time.January, 2, 0, 0, 0, 0, time.UTC)
	for day.Weekday() == time.Sunday {
		day = day.AddDate(0, 0, 1)
	}
	return day.Format("2006-01-02")
}
