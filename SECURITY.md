# Sicherheitsdokumentation

Dieses Dokument beschreibt die verbindlichen Sicherheitsanforderungen für den
Betrieb des Feature-Flag-Services und die zugrunde liegenden
Sicherheitsannahmen. Es ist Teil der Sicherheits- und
Datenschutzdokumentation und adressiert die Anforderungen an
Security-by-Design/-Default (EU Cyber Resilience Act) sowie an die
Transport- und Bindungskonfiguration.

## Unterstützte Deployment-Modelle

Der Dienst unterstützt zwei Betriebsmodi:

1. **Lokale Entwicklung (unsicher, nur Loopback).** Der Dienst bindet
   ausschließlich an `127.0.0.1` und spricht unverschlüsseltes HTTP. Dieses
   Modell ist ausschließlich für die lokale Entwicklung auf dem eigenen
   Rechner zulässig und darf nicht im Netzwerk exponiert werden.

2. **Produktion (TLS-terminierend).** Der Dienst wird hinter einem
   TLS-terminierenden Reverse Proxy betrieben (siehe unten) und ist nie
   direkt aus dem Internet erreichbar. Alternativ kann der Dienst selbst TLS
   über `http.ListenAndServeTLS` mit gültigem Zertifikat und privatem
   Schlüssel terminieren.

## Authentifizierung

Alle Endpunkte unter `/flags` sowie der Evaluierungs-Endpunkt
`GET /flags/{key}/evaluate` müssen geschützt werden. Es gilt:

- Der Zugriff erfolgt über einen geheimen API-Key aus der Umgebungsvariable
  **`FEATUREFLAGS_API_KEY`**.
- Der Vergleich des präsentierten Keys mit dem erwarteten Wert erfolgt in
  konstanter Zeit (`crypto/subtle.ConstantTimeCompare`), um Timing-Angriffe
  auszuschließen.
- Fehlender oder ungültiger Key wird mit `401 Unauthorized` als
  JSON-Fehlerobjekt beantwortet.
- Der Dienst **verweigert den Start**, wenn kein Key gesetzt ist, sodass er
  nicht versehentlich ungeschützt läuft.

Der Key selbst ist ein Geheimnis und darf nie im Repository, in
`RUN.json` (dort ausschließlich als `generate`/`external`) oder in der
Dokumentation hinterlegt werden.

## Transportverschlüsselung

Für den Produktionsbetrieb ist Transportverschlüsselung **verpflichtend**:

- **Empfohlener Weg:** Ein TLS-terminierender Reverse Proxy ist
  vorgeschaltet; der Dienst bindet ausschließlich an `127.0.0.1` und spricht
  internes, unverschlüsseltes HTTP.
- **Alternative:** Der Dienst terminiert TLS selbst über
  `http.ListenAndServeTLS` mit gültigem Zertifikat und privatem Schlüssel.
- Reines HTTP ist ausschließlich lokal an `127.0.0.1` zulässig (Entwicklung);
  ein HTTP-Binding auf öffentlichen oder nicht abgesicherten Interfaces ist
  nicht zulässig.

## Update- und Patch-Prozess

- Sicherheitsrelevante Änderungen werden über die reguläre Code-Review- und
  Merge-Pipeline ausgeliefert.
- Bekannte Schwachstellen in Abhängigkeiten werden geprüft, sobald externe
  Abhängigkeiten eingeführt werden (aktuell keine externen Abhängigkeiten;
  ausschließlich Go-Standardbibliothek).
- Ein Update erfordert einen erneuten Build (`go build ./...`) und den
  Neustart des Dienstes. Da der Store rein im Arbeitsspeicher liegt, gehen
  bei einem Neustart keine persistenten Daten verloren (es gibt keine).
- Sicherheitsrelevante Updates (z. B. der Go-Runtime) sind zeitnah nach
  Verfügbarkeit einzuspielen.

## Sicherheitsannahmen

Die folgenden Annahmen liegen dem Sicherheitsmodell zugrunde und sind beim
Betrieb sicherzustellen:

- Der Dienst läuft in einer vertrauenswürdigen, abgeschotteten Umgebung.
- Der Transportweg zum Dienst ist entweder TLS-terminiert (Reverse Proxy
  oder `ListenAndServeTLS`) oder auf das Loopback-Interface beschränkt.
- Der API-Key (`FEATUREFLAGS_API_KEY`) ist ein Geheimnis und wird nicht
  geteilt, nicht im Klartext übertragen und nicht in Logs oder im
  Repository gespeichert.
- `key` und `description` enthalten **keine personenbezogenen Daten**; der
  einzig zulässige Personenbezug ist der transitive `user`-Parameter der
  Evaluierung, der ausschließlich gehasht und nie gespeichert oder
  protokolliert wird.
- Die Logging-Middleware protokolliert ausschließlich Methode, Pfad ohne
  Query-String, Statuscode und Dauer; der `user`-Parameter erscheint in
  keinem Log-Eintrag.

## SBOM (Software Bill of Materials)

Zur Nachvollziehbarkeit der eingesetzten Komponenten wird eine SBOM erzeugt.
Da der Dienst ausschließlich die Go-Standardbibliothek nutzt, ist die SBOM
minimal, wird aber dennoch als Release-Artefakt bereitgestellt.

Erzeugung über die Modul-Liste:

```sh
go list -m -json all
```

Alternativ über die Build-Info des kompilierten Binaries:

```sh
go build -o featureflags .
go version -m featureflags
```

Die CI-Integration (automatische Erzeugung und Ablegung als
Release-Artefakt) ist nicht Teil dieses Dokumentationsstandes.
