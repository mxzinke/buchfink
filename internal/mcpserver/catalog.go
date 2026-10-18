package mcpserver

// registerTools registriert den Werkzeugkatalog. Was nur in der App geht
// (Festschreibung, Übermittlung, Pflicht erledigt, Aufstellen bis Offenlegen,
// Ergebnisverwendung, Stammdaten und Bankkonten, Darlehensverträge,
// Standardgruppe bestätigen, Prüfermodus, Apps, Regeln für Apps), hat kein
// Werkzeug; docs/entwicklung/mcp.md führt die Liste.
func registerTools(tb *Toolbox) {
	registerStatus(tb)
	registerAufgaben(tb)
	registerFristen(tb)
	registerKonten(tb)
	registerKontakte(tb)
	registerBelege(tb)
	registerEingang(tb)
	registerBuchungen(tb)
	registerBuchen(tb)
	registerBank(tb)
	registerPosten(tb)
	registerRechnung(tb)
	registerAbschluss(tb)
	registerErklaerung(tb)
}
