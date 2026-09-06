VERDICT: BUGS_FOUND

**Bug 1: Router verdrahtet Pfadparameter-Routen nicht (GET/PUT/DELETE /flags/{key} und Evaluierung liefern 404)**

- **Titel**: Router verdrahtet Pfadparameter-Routen nicht
- **Symptom**: Die Kern-API des Feature-Flag-Service ist teilweise nicht erreichbar: Einzelabruf, Update, Delete und Evaluierung eines Flags über die Pfade mit `{key}` antworten mit HTTP 404 statt der erwarteten Statuscodes (200/204). Dadurch sind AC-05, AC-06, AC-07, AC-08 und AC-11 nicht erfüllt – Flags können zwar angelegt und gelistet werden, aber nicht gezielt gelesen, geändert, gelöscht oder ausgewertet.
- **Repro**: `go test ./...` im Projektverzeichnis ausführen. Der Test `TestRoutesAreWired` im Paket `featureflags/internal/router` schlägt fehl.
- **Evidence**:
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
- **Suspected file(s)**: `internal/router/router.go` – alle vier fehlgeschlagenen Routen teilen sich denselben Mechanismus (Muster mit `{key}`). Da die Basis-Routen (`GET /flags`, `POST /flags`, `GET /healthz`) offenbar funktionieren, liegt der Fehler vermutlich in der Registrierung oder dem Matching der Pfadparameter-Muster, nicht in den einzelnen Handlern.
- **Severity**: high (Kernfunktionalität des Feature-Flag-Service ist blockiert; ohne diese Endpunkte ist der Service praktisch unbrauchbar).