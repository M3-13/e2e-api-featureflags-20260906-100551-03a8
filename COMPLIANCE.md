VERDICT: CHANGES_REQUESTED

## Prüfumfang

Projekttyp: `go-backend` — eine reine REST-API ohne Endnutzer-Weboberfläche. Damit entfallen Cookie-/Consent-Pflichten, Impressums-/Datenschutzerklärungs-Webseiten und WCAG/BITV/EAA-Anforderungen. Einschlägig sind dagegen die DSGVO, soweit personenbezogene Daten verarbeitet werden, und der Cyber Resilience Act, soweit die Software als Produkt mit digitalen Elementen in Verkehr gebracht wird. Der AI Act ist nicht einschlägig, da keine KI-Funktion erkennbar ist.

## Zusammenfassung

Die Implementierung ist insgesamt solide: Fail-closed-API-Key-Schutz, `ConstantTimeCompare`, Body-/Header-/Response-Limits, Server-Timeouts, Log-Sanitisierung, keine Query-String-Protokollierung, kein CORS und durchgängig JSON-`Content-Type` sprechen für Security-by-default. Die offenen Punkte betreffen vor allem Transportverschlüsselung, fehlenden Brute-Force-/Rate-Limit-Schutz sowie den CRA-Dokumentations- und Update-Nachweis. Diese Lücken sind behebbar; ein fundamentaler Verstoß liegt nicht vor.

---

## DSGVO

### DSGVO-1 — Schweregrad: hoch  
**Unverschlüsselte Übertragung von API-Key und `user`-Parameter**

Der `Authorization`-Header (`Bearer <API_KEY>`) und der `user`-Query-Parameter aus Evaluierungsanfragen werden als personenbezogene bzw. zugangsrelevante Daten verarbeitet. `main.go` startet den Server ausschließlich mit `server.ListenAndServe()` — also ohne TLS. Zwar ist der Default-Host `127.0.0.1` günstig, aber sobald `HOST=0.0.0.0` gesetzt wird, laufen Token und Nutzerkennung unverschlüsselt über das Netz. Das genügt nicht dem Schutzniveau nach Art. 32 DSGVO und ist zugleich ein CRA-Secure-by-default-Problem.

**Remedy:**  
`main.go` erweitern: Umgebungsvariablen `TLS_CERT_FILE` und `TLS_KEY_FILE` einlesen; wenn beide gesetzt sind, `server.ListenAndServeTLS(...)` verwenden. Falls TLS durch einen vorgelagerten Reverse Proxy terminiert wird, muss das in `SECURITY.md` und `README.md` als verbindliche Betriebsvoraussetzung festgeschrieben werden. Betriebsregel ergänzen: „Der Service darf nur über TLS-terminierende Endpunkte erreichbar sein; `Authorization`- und `user`-Parameter dürfen nicht unverschlüsselt übertragen werden.“ Zusätzlich sollte der `HOST`-Default `127.0.0.1` dokumentiert bleiben.

---

### DSGVO-2 — Schweregrad: mittel  
**Kein Rate-Limit/Brute-Force-Schutz für den API-Key**

Die Middleware `RequireAPIKey` vergleicht den Token zwar mit `crypto/subtle.ConstantTimeCompare`, setzt aber keinen Durchsatz- oder Fehlversuchs-Schutz um. Ein Angreifer mit Netzwerkzugriff kann den Schlüssel unbegrenzt online erraten. Das betrifft die Vertraulichkeit der verarbeiteten `user`-IDs und die Sicherheit des Dienstes nach Art. 32 DSGVO.

**Remedy:**  
Neue Datei `internal/middleware/ratelimit.go` einführen und in `main.go` um `RequireAPIKey` beziehungsweise `Logging` legen. Sinnvoll: Sliding-Window-Limit pro Client-IP oder Token-Hash, konfigurierbar über Env-Variablen wie `RATE_LIMIT_RPS` und `RATE_LIMIT_BURST`, zusätzlich exponentielle Verzögerung oder temporäre Sperre bei wiederholten 401/503-Antworten. Die Limits müssen so dimensioniert sein, dass legitime Flag-Verwaltung und Evaluierungsanfragen nicht blockiert werden. In `SECURITY.md` dokumentieren.

---

### DSGVO-3 — Schweregrad: niedrig  
**Log-Sanitisierung deckt nicht alle Unicode-Zeilenumbrüche ab**

Die Funktion `sanitize` in `internal/middleware/middleware.go` ersetzt nur `unicode.IsControl`. Unicode-Zeilen- und Absatztrenner U+2028/U+2029 (Kategorien Zl/Zp) können in manchen Log-Viewern als zusätzliche Zeilenumbrüche wirken. Das Risiko ist gering, aber für die Log-Integrität relevant.

**Remedy:**  
In `sanitize` die Bedingung erweitern:

```go
if unicode.IsControl(r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r) {
    b.WriteByte(' ')
    continue
}
```

---

### DSGVO-4 — Schweregrad: niedrig  
**Dokumentation von Rechtsgrundlage und vorgelagerten Logs**

Der `user`-Parameter ist eine pseudonyme Nutzerkennung und damit personenbezogen. Die Verarbeitung erfolgt transient und ohne Speicherung; das ist datenminimierend. Im sichtbaren Code fehlt aber eine explizite Zuordnung der Rechtsgrundlage. Auch muss sichergestellt sein, dass vorgelagerte Proxys den Query-String nicht in Logs aufnehmen.

**Remedy:**  
In `COMPLIANCE.md` einen DSGVO-Abschnitt ergänzen: Verarbeitung des `user`-Parameters ausschließlich zur deterministischen Flag-Evaluierung; Rechtsgrundlage z. B. Art. 6 Abs. 1 lit. b oder f DSGVO im Auftrag des Verantwortlichen; keine Speicherung des `user`-Werts; keine Protokollierung des Query-Strings durch den Dienst selbst. Betriebsvorgabe: Deep-Logging von Query-Strings in vorgelagerten Proxys deaktivieren oder maskieren.

---

### DSGVO-5 — Schweregrad: niedrig  
**Freitextfeld `description` kann unbeabsichtigt PII aufnehmen**

Das Datenmodell erlaubt in `description` beliebigen Text bis 500 Bytes. Der Store selbst ist nicht für personenbezogene Daten vorgesehen; der Betreiber muss die Verwendung von PII in Flag-Metadaten ausschließen. Es fehlt ein dokumentarischer Hinweis.

**Remedy:**  
In `README.md` oder `COMPLIANCE.md` aufnehmen: „`key` und `description` dürfen keine personenbezogenen Daten enthalten; der Flag-Store ist ausschließlich für Feature-Flag-Metadaten bestimmt.“ So bleibt der Store frei von PII und die Betroffenenrechte werden nicht berührt.

---

## Cyber Resilience Act (CRA)

### CRA-1 — Schweregrad: mittel  
**Fehlender sichtbarer Update-/Patch-Nachweis und fehlende Versionskennzeichnung**

Für Produkte mit digitalen Elementen verlangt der CRA eine identifizierbare Version und die Fähigkeit, Sicherheitsupdates zu verteilen. Im sichtbaren Code liefert `/healthz` nur `{"status":"ok"}`, ohne Versions- oder Build-Information. Ein Update-Prozess ist nicht aus dem Code ersichtlich.

**Remedy:**  
`internal/handlers/health.go` erweitern, sodass der Health-Endpunkt eine Build-Version liefert, z. B.:

```json
{"status":"ok","version":"<build-info>"}
```

Die Versions-/Build-Kennzeichnung in `main.go` per Linker-Flag oder Konstante befüllen. In `SECURITY.md` einen Abschnitt „Update-/Patch-Prozess“ ergänzen: wie Updates ausgeliefert, Sicherheitslücken gemeldet und behoben werden, inklusive Supportzeitraum.

---

### CRA-2 — Schweregrad: hoch  
**Transportverschlüsselung als Secure-by-default-Anforderung**

Siehe DSGVO-1. Der CRA verlangt Secure by Design und Secure by Default; ein Standardbetrieb über unverschlüsseltes HTTP mit API-Key und Nutzerkennung erfüllt das nicht, sobald der Dienst über eine nicht-lokale Schnittstelle erreichbar ist.

**Remedy:**  
Wie DSGVO-1: TLS-Auslieferung oder verbindliche TLS-Terminierung durch einen Reverse Proxy. In `SECURITY.md` als „verbindliche Betriebsvoraussetzung“ aufnehmen.

---

### CRA-3 — Schweregrad: niedrig  
**SBOM-/Abhängigkeitsnachweis nicht im sichtbaren Build-Prozess**

Das Projekt nutzt laut Code ausschließlich die Go-Standardbibliothek; `go.mod` ist vorhanden. Eine SBOM ist damit einfacher zu erzeugen, aber im sichtbaren Code/Releasepfad nicht erkennbar. Für CRA-Konformität bei Inverkehrbringen ist ein SBOM-Nachweis sinnvoll.

**Remedy:**  
In der Build-/Release-Pipeline eine SBOM im SPDX- oder CycloneDX-Format erzeugen. In `SECURITY.md` dokumentieren, dass die Abhängigkeitsliste durch `go.mod` abgebildet ist und dass im Release eine SBOM mitgeliefert wird.

---

### CRA-4 — Schweregrad: niedrig  
**Dokumentation der Sicherheitseigenschaften als Security by Design/Default**

Der Code setzt viele CRA-relevante Maßnahmen um: Fail-closed bei fehlendem API-Key, Body- und Header-Limits, Response-Limit, Timeouts, Log-Sanitisierung ohne PII, kein CORS. Diese Eigenschaften sollten ausdrücklich als dokumentierte Sicherheitsannahmen nachgewiesen werden.

**Remedy:**  
In `SECURITY.md` einen Abschnitt „Security by Design/Default“ ergänzen, der die vorhandenen Maßnahmen auflistet und ihre Wirkung beschreibt. Falls bereits enthalten, ist dieser Punkt als erfüllt anzusehen.

---

## AI Act

Nicht einschlägig. Die Software enthält keine KI-Funktion im Sinne der KI-Verordnung.

---

## Pflichttexte und Web-UI

Nicht einschlägig für diesen Projekttyp. Als reine REST-API ohne Endnutzer-UI bestehen keine Cookie-/Consent-, Impressums- oder Web-Datenschutzerklärungspflichten. Bei entgeltlicher oder geschäftlicher Bereitstellung an Kunden sind jedoch vertragliche Regelungen, insbesondere ein Auftragsverarbeitungsvertrag nach Art. 28 DSGVO, erforderlich.

---

## Barrierefreiheit

Nicht einschlägig. Es ist keine öffentliche Web-UI vorhanden.