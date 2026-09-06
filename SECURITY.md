VERDICT: BLOCKED

## Gesamteinschätzung

Der Code ist weitgehend sauber gegen typische Injection- und Eingabeprobleme abgesichert: Flag-Keys werden bei `POST /flags` auf einen sicheren Zeichensatz beschränkt, der Request-Body wird vor dem vollständigen Einlesen auf 1 MiB begrenzt, JSON-Antworten setzen durchgehend `Content-Type: application/json; charset=utf-8`, und die Logging-Middleware entfernt Steuerzeichen aus dem Pfad und protokolliert keine Query-Parameter. Es wurden keine Hardcoded Secrets oder externen Abhängigkeiten mit bekannten Schwachstellen gefunden. Der Scanner-Output ist leer und liefert daher keine zusätzlichen Befunde.

Dennoch ist das Produkt aktuell nicht sicher auslieferbar: Sämtliche REST-Endpunkte sind ohne jede Authentifizierung und Autorisierung zugänglich. Jeder Netzwerkteilnehmer kann Feature-Flags anlegen, auslesen, verändern und löschen und damit die Anwendung steuern bzw. sabotieren. Das ist ein Zugriffskontrollfehler mit hohem Risiko und führt zur Blockierung.

---

## Befunde

### 1. Fehlende Authentifizierung und Autorisierung (AuthN/AuthZ)
- **Schweregrad:** hoch (kritisch bei Internet-Exponierung)
- **Betroffene Stellen:**
  - `main.go:20-27` – Serveraufbau und Route-Handling ohne Zugriffsschutz
  - `internal/router/router.go:39-51` – Route-Mapping für alle öffentlichen Endpunkte
  - `internal/handlers/service.go:18-62` – zentraler Dispatcher ohne Auth-Prüfung
  - `internal/handlers/flags.go` – `CreateFlag`, `UpdateFlag`, `DeleteFlag` ungeschützt
  - `internal/handlers/evaluate.go` – Evaluierung ungeschützt
- **Beschreibung:**  
  Alle Endpunkte (`POST /flags`, `GET /flags`, `GET /flags/{key}`, `PUT /flags/{key}`, `DELETE /flags/{key}`, `GET /flags/{key}/evaluate`) sind ohne Authentifizierung erreichbar. Dadurch kann ein Angreifer:
  - Flags anlegen oder löschen (Veränderung der Anwendungslogik),
  - Rollout-Prozentsätze oder `enabled`-Status manipulieren,
  - Evaluierungsergebnisse für beliebige Nutzer abrufen,
  - den In-Memory-Store mit beliebigen Daten fluten.
- **Konkrete Lösung:**
  1. Eine Authentifizierungs-Middleware implementieren, die vor dem Router alle `/flags`-Routen schützt.
  2. Einen geheimen API-Key aus der Umgebungsvariable (z. B. `FEATUREFLAGS_API_KEY`) lesen und mit `crypto/subtle.ConstantTimeCompare` vergleichen. Alternativ mTLS, OAuth2 oder ein signiertes Token verwenden.
  3. Bei fehlendem oder ungültigem Key `401 Unauthorized` als JSON-Fehlerobjekt zurückgeben.
  4. Start verweigern, wenn kein Key gesetzt ist (`log.Fatal`), sodass der Dienst nicht versehentlich ungeschützt startet.
  5. Optional nach Lese-/Schreibrechten differenzieren: Lesende Endpunkte benötigen eine Leseberechtigung, mutierende Endpunkte eine Schreibberechtigung.

---

### 2. Fehlende HTTP-Server-Timeouts
- **Schweregrad:** mittel
- **Betroffene Stelle:** `main.go:25` – `http.ListenAndServe(addr, handler)`
- **Beschreibung:**  
  Es wird der Standard-`http.Server` ohne explizite Timeouts verwendet. Dadurch ist der Dienst anfällig für Slowloris-Angriffe und kann durch offene, langsame Verbindungen Ressourcen verlieren.
- **Konkrete Lösung:**
  ```go
  srv := &http.Server{
      Addr:              addr,
      Handler:           handler,
      ReadHeaderTimeout: 5 * time.Second,
      ReadTimeout:       10 * time.Second,
      WriteTimeout:      30 * time.Second,
      IdleTimeout:       60 * time.Second,
      MaxHeaderBytes:    1 << 20,
  }
  if err := srv.ListenAndServe(); err != nil {
      log.Fatal(err)
  }
  ```
  Dabei sicherstellen, dass legitime Clients mit normalen Latenzen weiterhin funktionieren; die Werte können je nach Einsatzumgebung angepasst werden.

---

### 3. Fehlende Transportverschlüsselung (HTTP statt HTTPS)
- **Schweregrad:** mittel
- **Betroffene Stelle:** `main.go:25`
- **Beschreibung:**  
  Der Server bindet ausschließlich auf unverschlüsseltes HTTP. Flag-Beschreibungen, Keys und Evaluierungsergebnisse können im Netzwerk mitgelesen oder manipuliert werden, insbesondere weil keine Authentifizierung auf Anwendungsebene vorhanden ist.
- **Konkrete Lösung:**
  - `http.ListenAndServeTLS` mit gültigem Zertifikat und privatem Schlüssel verwenden, **oder**
  - dokumentieren und betrieblich sicherstellen, dass ein TLS-terminierender Reverse Proxy vorgeschaltet ist und der Dienst nur an `127.0.0.1` gebunden wird.
  - Für Produktion ist Transportverschlüsselung verpflichtend; ein reines HTTP-Binding ist nur für lokale Entwicklung akzeptabel.

---

### 4. Unbegrenzte Antwortpufferung im Router
- **Schweregrad:** niedrig – mittel
- **Betroffene Stelle:** `internal/router/router.go:15-25` (`recorder`) und `56-66` (Pufferung)
- **Beschreibung:**  
  Jeder Response-Body wird vor dem Senden vollständig in einen `bytes.Buffer` geschrieben. Ein Angreifer oder autorisierter Nutzer kann durch viele oder sehr große Flags eine große `GET /flags`-Antwort erzeugen und dadurch den Speicherverbrauch des Prozesses erhöhen. In Kombination mit fehlender Authentifizierung kann dies zu einem einfachen DoS beitragen.
- **Konkrete Lösung:**  
  Den Router so umbauen, dass er 404/405-Fälle ohne vollständige Antwortpufferung erkennen kann, z. B. durch direkte Bindung der Handler an den `http.ServeMux` und eigene `NotFoundHandler`/`MethodNotAllowedHandler`. Alternativ im `recorder` eine maximale Body-Größe (z. B. 4 MiB) einbauen und bei Überschreitung mit `413` abbrechen. Die bestehenden Status- und JSON-Antwortformate müssen dabei erhalten bleiben.

---

### 5. Optional: Request-Content-Type validieren
- **Schweregrad:** niedrig
- **Betroffene Stelle:** `internal/handlers/flags.go:53-57` (`decodeJSON`)
- **Beschreibung:**  
  `POST /flags` und `PUT /flags/{key}` akzeptieren JSON-Bodies auch bei fehlendem oder abweichendem `Content-Type` (z. B. `text/plain`). Das ist aktuell kein direktes Exploit-Risiko, aber eine strikte Validierung reduziert die Angriffsfläche und verhindert Mehrdeutigkeiten.
- **Konkrete Lösung:**
  ```go
  ct := r.Header.Get("Content-Type")
  mt, _, err := mime.ParseMediaType(ct)
  if err != nil || mt != "application/json" {
      writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
      return
  }
  ```
  Diese Prüfung sollte in `decodeJSON` vor dem Body-Lesen erfolgen.

---

## Nicht beanstandet

- **Secrets:** Keine Hardcoded Keys, Passwörter oder Tokens gefunden.
- **Injection/Path Traversal:** Key-Validierung `^[a-zA-Z0-9_-]+$` verhindert Pfadtrenner und `..` bei der Anlage. GET/PUT/DELETE lesen lediglich aus der In-Memory-Map; kein Dateizugriff.
- **CORS:** Es werden keine `Access-Control-Allow-Origin`-Header gesetzt; AC-18 ist erfüllt.
- **Logging:** Pfad wird ohne Query-String und mit Steuerzeichen-Bereinigung protokolliert; der `user`-Parameter erscheint nicht im Log (AC-19, AC-21 erfüllt).
- **Dependencies:** Es werden ausschließlich Go-Standardbibliothekspakete verwendet; kein externes Dependency-Risiko erkennbar. Der Scanner-Output ist leer.