VERDICT: BUGS_FOUND

- **Titel**: Routen für `/flags/{key}` nicht verdrahtet – GET/PUT/DELETE/Evaluate liefern 404
- **Symptom**: Aus Nutzersicht sind alle Einzel-Flag-Endpunkte nicht erreichbar: `GET /flags/<key>`, `PUT /flags/<key>`, `DELETE /flags/<key>` und `GET /flags/<key>/evaluate` antworten mit 404. Damit können angelegte Flags nicht abgerufen, aktualisiert, gelöscht oder evaluiert werden – Kernfunktionalität der Spezifikation ist gebrochen.
- **Repro**: `go test ./...` ausführen, bzw. direkt `GET /flags/my-flag`, `PUT /flags/my-flag`, `DELETE /flags/my-flag`, `GET /flags/my-flag/evaluate` gegen den Server absetzen.
- **Beleg**:
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
  FAIL
  FAIL	featureflags/internal/router	0.342s
  ```
- **Verdächtige Datei(en)**: `internal/router/router.go` – alle vier fehlschlagenden Routen teilen dieselbe `{key}`-Musterform und werden gemeinsam in `router.New` registriert. Die identische Fehlerform spricht für eine gemeinsame Ursache in der Verdrahtung der Wildcard-Muster, nicht in den einzelnen Handlern.
- **Schweregrad**: high