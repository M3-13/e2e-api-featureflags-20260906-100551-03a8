VERDICT: CHANGES_REQUESTED

## Sicherheitsbewertung Feature-Flag-Service

Der Service ist insgesamt solide umgesetzt. Die im Sprint geforderten Sicherheits-AK (AC-16 bis AC-21) sind in der vorliegenden Implementierung erfüllt. Es wurden **keine kritischen Sicherheitslücken** festgestellt; mehrere Härtungsmaßnahmen sind jedoch empfehlenswert, insbesondere zur Ressourcenbegrenzung und Transportverschlüsselung.

Kein Scanner-Output vorhanden (`no applicable security scanners`). Es existieren keine externen Go-Abhängigkeiten; die Bewertung basiert auf der Codeanalyse.

---

### Befunde

#### 1. Mittel — Unbegrenzte Flag-Anzahl und fehlende Key-Längenbegrenzung (Ressourcenerschöpfung)
- **Betroffen:** `internal/handlers/flags.go`, `internal/store/store.go`
- **Beschreibung:** Der Flag-Key wird nur auf den erlaubten Zeichensatz geprüft, nicht jedoch auf eine Länge. Da der Request-Body auf 1 MiB begrenzt ist, kann ein einzelner Key bis zu ca. 1 MiB groß werden. Zusätzlich gibt es kein Limit für die Gesamtzahl der Flags. Ein Angreifer mit gültigem API-Key (oder ein kompromittierter Client) kann den In-Memory-Store durch viele große Keys/Flags unbegrenzt auffüllen. `GET /flags` puffert die Antwort dann bis zum Router-Limit von 4 MiB ab, aber der Speicherverbrauch bleibt bestehen.
- **Fix:**
  - In `CreateFlag` nach der `keyPattern`-Prüfung eine maximale Key-Länge einführen, z. B. 128 Bytes:
    ```go
    const maxKeyLength = 128
    if len(req.Key) > maxKeyLength {
        writeError(w, http.StatusBadRequest, "key too long")
        return
    }
    ```
  - Im Store eine maximale Flag-Anzahl erzwingen, z. B.:
    ```go
    var ErrCapacity = errors.New("flag capacity reached")
    const maxFlags = 10000

    func (s *Store) Create(f Flag) error {
        // ...
        if _, exists := s.flags[f.Key]; !exists && len(s.flags) >= maxFlags {
            return ErrCapacity
        }
        // ...
    }
    ```
    `ErrCapacity` in `CreateFlag` auf einen passenden 5xx-Fehler abbilden.
- **Reconciliation:** Legitime Keys bleiben bei 128 Bytes klar möglich; die Anzahlgrenze ist hoch genug für den vorgesehenen In-Memory-Dienst.

#### 2. Mittel — Transport unverschlüsselt, Bearer-Token und Flag-Daten im Klartext
- **Betroffen:** `main.go`
- **Beschreibung:** Der Server lauscht ausschließlich auf HTTP. Sobald `HOST=0.0.0.0` bzw. ein extern erreichbares Interface gesetzt wird, können der `Authorization: Bearer`-Header und die Flag-Konfiguration im Netzwerk mitgelesen werden. Der Default `127.0.0.1` mindert das Risiko, hebt es aber für produktive Deployments nicht auf.
- **Fix:** Optionale TLS-Unterstützung ergänzen:
  ```go
  certFile := os.Getenv("TLS_CERT_FILE")
  keyFile := os.Getenv("TLS_KEY_FILE")
  if certFile != "" && keyFile != "" {
      log.Printf("featureflags listening on %s (TLS)", addr)
      err = server.ListenAndServeTLS(certFile, keyFile)
  } else {
      log.Printf("WARNING: serving HTTP without TLS; deploy only behind a TLS-terminating proxy or in a private network")
      err = server.ListenAndServe()
  }
  ```
- **Reconciliation:** Lokale Entwicklung ohne TLS bleibt möglich; für produktive/externe Deployments kann TLS aktiviert werden, ohne die Produktfunktion zu ändern.

#### 3. Niedrig — Pfad-Normalisierung vor API-Key-Prüfung verwenden
- **Betroffen:** `internal/middleware/middleware.go`, Funktion `RequireAPIKey`
- **Beschreibung:** Die Prüfung `path != "/flags" && !strings.HasPrefix(path, "/flags/")` verwendet den rohen `r.URL.Path`. Pfade wie `//flags` oder `/%2Fflags` umgehen dadurch rein syntaktisch die API-Key-Prüfung. Über den vorgeschalteten Router wird der Handler dabei aktuell nicht aufgerufen, sondern lediglich eine Redirect-/404-Antwort erzeugt; ein direkter Datenzugriff entsteht dadurch nicht. Es ist aber gefährlich, Schutzentscheidung und Routing auf unterschiedliche Pfadnormalisierungen zu stützen.
- **Fix:**
  ```go
  p := path.Clean(r.URL.Path)
  if p != flagsPath && !strings.HasPrefix(p, flagsPath+"/") {
      next.ServeHTTP(w, r)
      return
  }
  ```
  `path` aus der Standardbibliothek importieren.
- **Reconciliation:** `/flags`, `/flags/{key}` und `/flags/{key}/evaluate` funktionieren weiterhin; `/healthz` bleibt offen. Andere, normalisiert außerhalb von `/flags` liegende Pfade bleiben ungeschützt, wie vorgesehen.

#### 4. Niedrig — `X-Content-Type-Options: nosniff` fehlt
- **Betroffen:** `internal/handlers/respond.go` (`writeJSON`), `internal/middleware/middleware.go` (`writeError`)
- **Beschreibung:** JSON-Antworten setzen zwar korrekt `Content-Type: application/json; charset=utf-8`, aber der Browser-Header `X-Content-Type-Options: nosniff` fehlt. Das ist ein zusätzlicher MIME-Sniffing-Schutz bei alten Browsern.
- **Fix:** In beiden zentralen Response-Funktionen ergänzen:
  ```go
  w.Header().Set("X-Content-Type-Options", "nosniff")
  ```
- **Reconciliation:** Hat keine Auswirkung auf die Funktionalität.

#### 5. Niedrig — Kein Rate-Limiting vorhanden
- **Betroffen:** `main.go` / Middleware-Schicht
- **Beschreibung:** Es gibt keine Begrenzung der Anfragefrequenz. In Kombination mit dem In-Memory-Store kann ein einzelner Client durch viele Anfragen CPU und I/O belasten. Da der API-Key als Authentifizierung dient, sollte dieser Schutz idealerweise pro Token konfigurierbar sein.
- **Fix:** Optional eine konfigurierbare Rate-Limit-Middleware (z. B. Token-Bucket pro Client-Adresse/Tenant) einführen, mit großzügigen Standardwerten und Konfiguration über Umgebungsvariablen.
- **Reconciliation:** Die Standardwerte müssen für den legitimen Evaluierungsbetrieb ausreichend dimensioniert sein; das Limit darf ruhig hoch sein oder deaktivierbar bleiben, um Produktivlast nicht zu brechen.

---

### Positiv geprüfte Bereiche

- **Secrets:** Keine hartkodierten Schlüssel. Der API-Key kommt aus `FEATUREFLAGS_API_KEY` und wird mit `crypto/subtle.ConstantTimeCompare` verglichen. Fail-Closed-Verhalten bei fehlendem Key (`503 service locked`) ist sauber.
- **Injection/Inputs:** JSON-Content-Type-Prüfung, Body-Größenlimit 1 MiB, Key-Whitelist `[a-zA-Z0-9_-]+`, Rollout-Validierung 0–100, Description-Längenlimit 500 Bytes, keine SQL-/Pfadinjektion erkennbar.
- **AuthN/AuthZ:** `RequireAPIKey` schützt alle `/flags`-Routen; `/healthz` bleibt bewusst offen. Bearer-Token-Prüfung ist konstantzeitvergleichend.
- **Dependencies:** Keine externen Dependencies im Modul; keine anfälligen Pakete sichtbar.
- **Konfiguration/Transport:** Server-Timeoutwerte und `MaxHeaderBytes` sind gesetzt; keine unsicheren CORS-Header; Logging protokolliert keine Query-Strings und keine `user`-Werte; Steuerzeichen im Pfad werden neutralisiert.