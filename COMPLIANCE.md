VERDICT: CHANGES_REQUESTED

Geprüft wird der vollständig gemergte Produktstand des `go-backend`. Reine REST-API ohne Endnutzer-UI: Pflichttexte, Cookie-/Consent-Banner und die Barrierefreiheitsanforderungen für öffentliche Web-UI sind daher nicht einschlägig. Relevant sind v. a. DSGVO, CRA und allgemeine Sicherheits-/Datenschutzanforderungen.

## 1. DSGVO / Datenschutz

### 1.1 Hoch: Unauthentifizierte, unverschlüsselte Admin-artige CRUD-API
`main.go` bindet mit `":" + port` an alle Interfaces und verwendet `http.ListenAndServe` — also HTTP ohne TLS. Die API erlaubt ohne Authentifizierung das Anlegen, Auslesen, Ändern und Löschen von Flags inklusive des Freitextfeldes `description`. Wenn dort personenbezogene Daten landen, sind sie für jeden im Netz erreichbar und werden im Klartext übertragen. Das betrifft Art. 5 Abs. 1 lit. f, Art. 25 und Art. 32 DSGVO.

**Abhilfe:**  
- `main.go`: Default-Bindung auf `127.0.0.1` setzen, z. B. `host := os.Getenv("HOST"); if host == "" { host = "127.0.0.1" }`, sofern kein expliziter Deployment-Modus gewählt wird.  
- TLS entweder direkt (`http.ListenAndServeTLS`) oder dokumentiert per vorgeschaltetem TLS-Terminierungs-Proxy umsetzen.  
- Eine Authentifizierung (z. B. statisches Token oder API-Key-Middleware) vor die mutierenden/lesenden Admin-Endpunkte schalten; andernfalls die Auslieferung nur im internen, abgesicherten Netz dokumentieren.

### 1.2 Mittel: Keine sichtbare Rechtsgrundlage und kein Verarbeitungskonzept für `user` und Flag-Inhalte
`internal/handlers/evaluate.go` verarbeitet den Query-Parameter `user` als Personenkennung. Der Wert wird zwar nicht geloggt und nicht gespeichert, aber die Verarbeitung als solche benötigt eine dokumentierte Rechtsgrundlage (z. B. Art. 6 Abs. 1 lit. b oder f DSGVO bzw. Auftragsverarbeitung). Auch für die in `internal/store/store.go` gehaltenen `key`/`description`-Felder fehlt ein sichtbares Verarbeitungs- und Löschkonzept.

**Abhilfe:**  
- `README.md` oder neue `SECURITY.md` ergänzen: Zweck der Verarbeitung, Rechtsgrundlage, Datenkategorien, Speicherdauer (Prozesslaufzeit, kein persistentes Speichern), Hinweis, dass `user` nur transient gehasht wird.  
- `AGENTS.md` oder `README.md`: betriebliche Vorgabe aufnehmen, dass `description` und `key` keine personenbezogenen Daten enthalten dürfen.

### 1.3 Mittel: `description` ist unvalidierter Freitext mit bis zu 1 MiB
Das Feld `description` in `internal/handlers/flags.go` wird nur durch das Body-Limit begrenzt, nicht aber auf Länge, Zeichenklasse oder Zweckbindung geprüft. Das ist ein Risiko für Datenminimierung und unkontrollierte Speicherung personenbezogener Daten.

**Abhilfe:**  
- In `internal/handlers/flags.go` eine sinnvolle Maximallänge für `description` einführen (z. B. 500 Zeichen) und bei Überschreitung mit `400` antworten.  
- Alternativ dokumentieren und betrieblich absichern, dass `description` rein technisch und PII-frei ist.

### 1.4 Positiv
- `internal/middleware/middleware.go` loggt ausweislich des sichtbaren Codes nur Methode, Pfad ohne Query-String, Statuscode und Dauer. Der `user`-Parameter aus `GET /flags/{key}/evaluate?user=...` erscheint nicht im Log.  
- `sanitize` in derselben Datei neutralisiert C0-Steuerzeichen, sodass klassische Log-Injection über `\n`, `\r` und `\t` unterbunden wird.  
- `DELETE /flags/{key}` erlaubt immerhin die Löschung gespeicherter Flag-Daten.

## 2. EU Cyber Resilience Act (CRA)

### 2.1 Hoch: Security by Design/Default nicht vollständig erfüllt
Der Dienst startet ohne Authentifizierung, ohne TLS und lauscht standardmäßig auf allen Interfaces. Damit sind die CRA-Anforderungen an sichere Grundeinstellungen und Zugriffsschutz für ein Produkt mit digitalen Elementen nicht erfüllt.

**Abhilfe:**  
- Wie unter 1.1: `main.go` auf Loopback binden, TLS/Auth umsetzen oder Reverse-Proxy-Lösung dokumentieren.  
- In `README.md`/`SECURITY.md` die sichere Betriebsarchitektur verbindlich beschreiben.

### 2.2 Mittel: Keine sichtbare Sicherheitsdokumentation, kein Update-/Patch-Konzept, kein SBOM
Die einsehbaren Quellen enthalten keinen dokumentierten Sicherheitsarchitektur-Abschnitt, keine dokumentierten Security Properties und kein SBOM. `go.mod` ist vorhanden und offenbar ohne externe Abhängigkeiten, aber die CRA-Dokumentationspflichten sind im sichtbaren Stand nicht abgedeckt.

**Abhilfe:**  
- `README.md` oder neue `SECURITY.md` ergänzen: unterstützte Deployment-Modelle, Authentifizierungs-/Transportanforderungen, Update- und Patch-Prozess, gemeldete Sicherheitsannahmen.  
- SBOM in der CI erzeugen, z. B. mit `go list -m -json all` oder `go version -m`; Ergebnis als Release-Artefakt ablegen.  
- Bei künftigen externen Abhängigkeiten `go.sum` plus automatisierte Dependency-/Vulnerability-Prüfung vorsehen.

### 2.3 Niedrig: Log-Sanitize deckt nur C0-Steuerzeichen ab
`internal/middleware/middleware.go` prüft nur `r < 0x20 || r == 0x7f`. Manche Log-Parser behandeln auch C1-Steuerzeichen wie U+0085 als Zeilenumbruch.

**Abhilfe:**  
- `sanitize` in `internal/middleware/middleware.go` auf `unicode.IsControl` umstellen oder zusätzlich alle Zeichen mit `unicode.IsControl(r)` durch ein Leerzeichen ersetzen.

### 2.4 Positiv
- Keine externen Abhängigkeiten sichtbar; die Angriffsfläche durch Drittanbieter-Bibliotheken ist gering.  
- Request-Body-Limit von 1 MiB in `internal/handlers/flags.go` ist Security-by-Default.  
- Flag-`key` wird in `internal/handlers/flags.go` per `^[a-zA-Z0-9_-]+$` auf einen sicheren Zeichensatz beschränkt; Path-Traversal über den Key ist damit wirksam unterbunden.  
- JSON-Antworten setzen durchgängig `Content-Type: application/json; charset=utf-8`.  
- Kein `Access-Control-Allow-Origin` und keine vom Request-Origin übernommene CORS-Origin sichtbar.

## 3. EU AI Act

Kein KI-Modell, kein automatisiertes Entscheidungssystem im Sinne des AI Act sichtbar. Der Hash-Algorithmus in `internal/hash/hash.go` ist deterministische Feature-Zuordnung, keine KI. Der AI Act ist damit nicht einschlägig.

**Abhilfe:** keine erforderlich.

## 4. Pflichttexte & UI

Keine öffentliche Endnutzer-UI, kein Cookie-Setzen, kein Verkaufs-Flow. Impressum, Datenschutzerklärung, Cookie-Banner und Widerrufsbelehrung sind für diesen Projekttyp nicht direkt erforderlich.

**Abhilfe:** keine unmittelbar; entsprechende Texte wären erst beim Betreiben eines öffentlich erreichbaren Frontends oder bei direktem Endnutzerkontakt erforderlich.

## 5. Barrierefreiheit

Keine öffentliche Web-UI. WCAG/BITV/EAA sind daher für den sichtbaren Stand nicht anwendbar.

**Abhilfe:** keine erforderlich.