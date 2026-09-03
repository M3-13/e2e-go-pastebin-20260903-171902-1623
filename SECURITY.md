VERDICT: CHANGES_REQUESTED

## Sicherheitsbericht

### Bewertungsgrundlage
- Projekttyp: Go-Backend, reine Standardbibliothek (`net/http`), In-Memory-Speicher mit `sync.RWMutex`.
- Es wurde kein Sicherheitsscanner ausgeführt (laut Ausgabe: „no applicable security scanners for this project type“). Die Bewertung basiert daher auf manueller Codeanalyse.
- Der aktuelle Stand enthält den POST-Erstell-Endpunkt nur als Stub (`handler_create.go`). Dadurch sind zentrale Sicherheitsanforderungen aus der Sprint-Spezifikation derzeit nicht umgesetzt.

---

### 1. Secrets / geheimnisvolle Daten
**Keine Befunde.**  
Im sichtbaren Code wurden keine Hardcoded Secrets, Passwörter, Token oder privaten Schlüssel gefunden. Es erfolgt keine Protokollierung von Paste-Inhalten oder Request-Bodies.

---

### 2. Injection & Eingaben
**Aktuell keine ausnutzbaren Eingabeverarbeitungspfade.**  
Der Pfadparameter `id` wird ausschließlich als Schlüssel für die interne Map verwendet (`r.PathValue("id")` → `a.store.pastes[id]`). Dies ist bei einer reinen In-Memory-Map unkritisch. Der POST-Endpunkt, der Nutzereingaben entgegennehmen würde, ist derzeit nicht implementiert und antwortet mit `501`.

**Sicherheitslücke durch fehlende POST-Implementierung**  
**Severity: medium**  
**Datei:** `handler_create.go`  
**Beschreibung:** Der Handler `handleCreate` liefert immer `501 Not Implemented`. Damit sind die in der Spezifikation geforderten Sicherheitsmaßnahmen nicht umgesetzt: maximale Request-Body-Größe (AC-11), Validierung von `expires_in_seconds` (AC-13) und ID-Erzeugung mit `crypto/rand` (AC-14). Aktuell ist kein direkter Angriff möglich, weil keine Nutzereingaben verarbeitet werden. Vor Auslieferung ist der Endpunkt jedoch zu implementieren und abzusichern.  
**Konkrete Lösung:**
- Body-Größe vor dem vollständigen Einlesen mit `http.MaxBytesReader(w, r.Body, 1<<20)` auf 1 MiB begrenzen; bei Überschreitung mit `413` antworten.
- JSON dekodieren, leeren `content` mit `400` ablehnen.
- `expires_in_seconds` als positive Ganzzahl validieren und auf maximal 10 Jahre begrenzen.
- ID mit `crypto/rand` erzeugen, z. B. 16 Hex-Zeichen (16 Bytes Zufall → 128 Bit, erfüllt die Mindestanforderung von 64 Bit).

---

### 3. Authentifizierung / Autorisierung

#### S1: Öffentliche Lese- und Löschzugriffe (Broken Access Control)
**Severity:** medium  
**Datei:** `main.go`, `handler_delete.go`  
**Beschreibung:** Alle Routen (`GET /pastes`, `GET /pastes/{id}`, `DELETE /pastes/{id}`) sind ohne jegliche Authentifizierung oder Autorisierung erreichbar. Jeder, der eine gültige Paste-ID kennt, kann den Paste lesen oder löschen. Die ID ist zwar (falls korrekt implementiert) zufällig und schwer zu erraten, sobald sie jedoch bekannt wird (z. B. durch Weitergabe, Logs oder Referrer), kann ein Dritter den Paste unwiderruflich löschen.  
**Konkrete Lösung:**
- Für Mutationen (insbesondere `DELETE`) eine Authentifizierung einführen, z. B. einen Bearer-Token/API-Key, oder einen beim Erstellen vergebenen „Owner-Token“ verwenden.
- Alternativ `DELETE` vollständig entfernen, wenn kein Löschen erforderlich ist.
- Mindestens dokumentieren, dass die API öffentlich ist, und sicherstellen, dass keine IDs in Logs oder Referrer auftauchen.

---

### 4. Abhängigkeiten
**Keine Befunde.**  
Das Projekt verwendet ausschließlich die Go-Standardbibliothek. Es sind keine externen Abhängigkeiten mit bekannten Sicherheitslücken vorhanden.

---

### 5. Konfiguration & Transport

#### S2: Fehlende Timeouts am HTTP-Server
**Severity:** medium  
**Datei:** `main.go`  
**Beschreibung:** Der `http.Server` wird ohne `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout` oder `IdleTimeout` konfiguriert. Dies macht den Server anfällig für Slowloris-Angriffe und ermöglicht es, Verbindungen unbegrenzt offen zu halten und Ressourcen zu binden.  
**Konkrete Lösung:**
```go
server := &http.Server{
    Addr:              ":8080",
    Handler:           api.routes(),
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       15 * time.Second,
    WriteTimeout:      15 * time.Second,
    IdleTimeout:       60 * time.Second,
}
```
Dabei darauf achten, dass legitime, langsame Clients nicht durch zu aggressive Timeouts ausgeschlossen werden; die Werte können angepasst werden.

#### S3: Keine Transportverschlüsselung (TLS)
**Severity:** medium  
**Datei:** `main.go`  
**Beschreibung:** Der Server lauscht auf `:8080` ohne TLS. Paste-Inhalte können im Klartext über das Netzwerk abgehört werden. Gerade bei Pastebin-Diensten können Inhalte sensibel sein.  
**Konkrete Lösung:**
- TLS direkt im Server aktivieren (`server.ListenAndServeTLS` mit Zertifikat und Schlüssel) oder
- den Dienst hinter einen Reverse Proxy (z. B. nginx, Caddy) legen, der TLS terminiert.
- Bei Verwendung von Zertifikaten auf sichere Ablage und Rotation achten.

#### S4: Keine Begrenzung der Anzahl gespeicherter Pastes (Speicher-DoS)
**Severity:** low (aktuell nicht ausnutzbar, da POST nicht implementiert)  
**Datei:** `store.go`  
**Beschreibung:** Es existiert kein Limit für die Anzahl der in der Map gehaltenen Pastes oder die Gesamtgröße. Sobald der POST-Endpunkt implementiert ist, könnte ein Angreifer durch massenhaftes Anlegen von Pastes Speicher und Ressourcen erschöpfen.  
**Konkrete Lösung:**
- Maximale Anzahl Pastes definieren (z. B. 10.000) und bei Erreichen des Limits älteste abgelaufene Einträge entfernen oder neue Requests mit `503` ablehnen.
- Optional zusätzlich ein Gesamtspeicherlimit einführen.

---

### 6. Datenschutz

**Positiv geprüft:**
- `GET /pastes/{id}` setzt den Header `Cache-Control: no-store` (AC-16 erfüllt).
- Abgelaufene Pastes werden bei Zugriff über `GET /pastes/{id}`, `GET /pastes` und `DELETE /pastes/{id}` dauerhaft entfernt (AC-15 erfüllt).
- Es werden keine Paste-Inhalte oder vollständigen Request-Bodies protokolliert (AC-17 erfüllt).
- Fehlerantworten enthalten ausschließlich das Feld `error` mit generischen Meldungen ohne Stacktrace oder interne Details (AC-12 erfüllt).

---

### 7. Weitere Härtungsempfehlungen (low)

#### S5: Fehlender `X-Content-Type-Options`-Header
**Severity:** low  
**Datei:** `response.go`  
**Beschreibung:** Der Header `X-Content-Type-Options: nosniff` wird nicht gesetzt. Bei reinen JSON-APIs ist das Risiko gering, aber der Header verhindert MIME-Sniffing im Browser.  
**Konkrete Lösung:** In `writeJSON` vor `WriteHeader` zusätzlich `w.Header().Set("X-Content-Type-Options", "nosniff")` setzen.

#### S6: Standard-Go-Server-Header
**Severity:** low  
**Datei:** `main.go`  
**Beschreibung:** Go setzt standardmäßig einen `Server`-Header, der Versionsinformationen preisgibt (z. B. `Go-http-client/1.1`). Dies erleichtert Angreifern die Identifikation der Technologie.  
**Konkrete Lösung:** Einen Reverse Proxy verwenden, der den Header entfernt, oder im Handler den Header bewusst auf einen generischen Wert setzen/unterdrücken.

---

## Zusammenfassung
Das Produkt enthält **keine hochkritischen oder kritischen Schwachstellen** (keine Hardcoded Secrets, keine Injection/RCE, keine ausgenutzten CVEs, kein direkter PII-Leak). Es bestehen jedoch mehrere **mittlere Sicherheitslücken**: fehlende Authentifizierung für Löschoperationen, fehlende HTTP-Timeouts, keine Transportverschlüsselung sowie die unvollständige POST-Implementierung mit offenen Sicherheitsanforderungen. Diese Punkte sollten vor Auslieferung behoben werden, daher **CHANGES_REQUESTED**.