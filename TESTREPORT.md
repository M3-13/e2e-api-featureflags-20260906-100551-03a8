VERDICT: BUGS_FOUND

**Bug 1**

- **Title:** Routing zu `/flags/{key}` liefert 404 für gültige Schlüssel – Kernrouten nicht verdrahtet
- **Symptom:** Die zentralen Feature-Flag-Endpunkte `GET /flags/{key}`, `PUT /flags/{key}`, `DELETE /flags/{key}` und `GET /flags/{key}/evaluate` sind für einen gültigen Schlüssel wie `my-flag` nicht erreichbar. Sie antworten mit `404` statt der erwarteten Erfolgsantworten. Dadurch sind mehrere Acceptance-Kriterien (AC-05, AC-06, AC-07, AC-08/AC-11) zur Laufzeit nicht erfüllbar.
- **Repro:** `go test ./...` ausführen – der Test `TestRoutesAreWired` im Paket `featureflags/internal/router` schlägt fehl.
- **Evidence:**
  ```
  --- FAIL: TestRoutesAreWired (0.00s)
      --- FAIL: TestRoutesAreWired/get_flag (0.00s)
          routes_test.go:43: GET /flags/my-flag is not wired: got 404
      --- FAIL: TestRoutesAreWired/update_flag (0.00s)
          routes_test.go:43: PUT /flags/my-flag is not wired: got 404
      --- FAIL: TestRoutesAreWired/delete_flag (0.00s)
          routes_test.go:43: DELETE /flags/my-flag is not wired: got 404
      --- FAIL: TestRoutesAreWired/evaluate_flag (0.00s)
          routes_test.go:43: GET /flags/my-flag/evaluate is not wired: got 404
  ```
- **Suspected file(s):** Gemeinsame Ursache in der Routenregistrierung bzw. im Routing-Handler – primär `internal/router/router.go` im Zusammenspiel mit `internal/handlers/service.go`. Die vier fehlgeschlagenen Untertests teilen dieselbe Fehlerform (alle Routen unter `/flags/{key}` liefern 404), daher liegt der Fehler vermutlich zentral in der Verdrahtung der Muster/Handler und nicht in den einzelnen Handler-Dateien.
- **Severity:** high