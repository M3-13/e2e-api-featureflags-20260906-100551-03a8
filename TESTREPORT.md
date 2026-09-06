VERDICT: BUGS_FOUND

**Bug 1**
- **Titel:** Kern-Routen `/flags/{key}` und `/flags/{key}/evaluate` antworten mit 404
- **Symptom:** Die laut Spezifikation geforderten Endpunkte zum Abrufen (AC-05), Aktualisieren (AC-06), Löschen (AC-07) und Evaluieren (AC-08) eines Flags sind nicht erreichbar. Ein Nutzer erhält für `GET /flags/my-flag`, `PUT /flags/my-flag`, `DELETE /flags/my-flag` und `GET /flags/my-flag/evaluate` jeweils den Status 404, obwohl die Flags im Store angelegt wurden und die Handler isoliert funktionieren. Damit ist ein zentraler Teil der Feature-Flag-API zur Laufzeit defekt.
- **Repro:** `go test ./...` — der Integrationstest `TestRoutesAreWired` schlägt fehl; dieselben Routen liefern auch bei einem echten Serverstart 404.
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
- **Suspected file(s):** Die vier Fehler haben dieselbe Form und betreffen alle Routen mit `{key}`-Segment. Gemeinsame Ursache ist sehr wahrscheinlich die Routenverdrahtung bzw. die Delegation der Wildcard-Pfade im Router (`internal/router/router.go`) oder die nachgelagerte Pfadauflösung in `internal/handlers/service.go`, die für diese Methoden auf den 404-Pfad führt. Nicht lokalisiert auf einen einzelnen Handler, da alle `{key}`-Routen identisch scheitern.
- **Severity:** high