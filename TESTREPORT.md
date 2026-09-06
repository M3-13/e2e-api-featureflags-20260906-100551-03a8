VERDICT: BUGS_FOUND

**Bug-Liste**

- **Title:** Feature-Flag-Routen mit `{key}` sind nicht verdrahtet und liefern 404  
- **Symptom:** Die Kern-Endpunkte `GET /flags/{key}`, `PUT /flags/{key}`, `DELETE /flags/{key}` und `GET /flags/{key}/evaluate` sind über den HTTP-Router nicht erreichbar. Nutzer können ein per `POST /flags` angelegtes Flag nicht einzeln abrufen, aktualisieren, löschen oder evaluieren — alle vier Routen antworten mit `404`, obwohl sie laut Spezifikation funktionieren müssen.  
- **Repro:** `go test ./...` bzw. `go test ./internal/router` ausführen; der Test `TestRoutesAreWired` schlägt fehl. Manuell z. B. `GET /flags/my-flag` oder `GET /flags/my-flag/evaluate?user=alice` aufrufen.  
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
- **Suspected file(s):** `internal/router/router.go` (Registrierung/Weitergabe der `{key}`-Muster) und/oder `internal/handlers/service.go` (Auswertung von `r.Pattern`). Da alle parameterisierten Routen gemeinsam ausfallen, einfache Routen wie `/healthz` und `/flags` im Test aber nicht beanstandet werden, liegt die Ursache vermutlich in der Behandlung der `{key}`-Pfadmuster bzw. der Weitergabe von `r.PathValue`/`r.Pattern` an den Handler.  
- **Severity:** high