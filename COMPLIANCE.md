VERDICT: CHANGES_REQUESTED

Der geprüfte Stand ist rechtlich nicht blockiert, aber nicht marktreif: Der zentrale Anlege-Endpunkt `POST /pastes` ist nur ein Stub und erfüllt damit zentrale Sicherheits- und Datenschutzanforderungen nicht. Daneben bestehen Härtungslücken nach CRA und DSGVO.

## 1. DSGVO / GDPR

### 1.1 Fehlende Speicherbegrenzung / keine Standard-Ablaufzeit — **hoch**

**Befund:**  
`store.go` speichert `Paste`-Objekte ohne Ablaufzeit dauerhaft im Speicher. `expires_in_seconds` ist laut Spec optional; wird es weggelassen, bleibt der Paste bis zum manuellen Löschen oder Prozessneustart unbegrenzt gespeichert. Das verstößt gegen den Grundsatz der Speicherbegrenzung (Art. 5 Abs. 1 lit. e DSGVO), sobald Paste-Inhalte personenbezogene Daten enthalten.

**Konkrete Abhilfe:**  
- In der zu implementierenden `handleCreate`-Logik (`handler_create.go`) eine **Standard-Ablaufzeit** setzen, z. B. 24 Stunden oder 7 Tage, wenn `expires_in_seconds` fehlt.  
- Alternativ eine dokumentierte **maximale Aufbewahrungsdauer** einführen und im Code durchsetzen, z. B. indem `ExpiresAt` immer gesetzt wird.  
- Diese Default-Dauer in `README.md` dokumentieren.

### 1.2 Rechtsgrundlage und Datenschutzdokumentation — **mittel**

**Befund:**  
Das Backend verarbeitet potenziell personenbezogene Daten (beliebige Paste-Inhalte). Im sichtbaren Code/Repository ist keine Rechtsgrundlage nach Art. 6 DSGVO dokumentiert. Auch wenn die aktive Nutzung durch den Endnutzer eine Vertragserfüllung (Art. 6 Abs. 1 lit. b DSGVO) nahelegt, fehlt eine entsprechende Datenschutzinformation für den Betreiber.

**Konkrete Abhilfe:**  
- In `README.md` oder `AGENTS.md` einen Abschnitt „Datenschutz / Rechtsgrundlage“ ergänzen, der die Verarbeitung, Rechtsgrundlage, Löschfristen und Betroffenenrechte beschreibt.  
- Da es sich um ein Backend ohne UI handelt, ist keine eingebaute Datenschutzerklärung erforderlich; der Betreiber muss die Hinweise jedoch außerhalb der API bereitstellen.

### 1.3 Positiv erfüllte Datenschutzanforderungen

- `handler_read.go` setzt `Cache-Control: no-store` auch bei 404.  
- `handler_list.go` liefert nur Metadaten ohne `content` (Datenminimierung).  
- Abgelaufene Pastes werden bei Zugriff über `GET`, `LIST` und `DELETE` entfernt.  
- Es findet keine Protokollierung von Paste-Inhalten oder Request-Bodies statt.  
- Fehlerantworten enthalten nur generische Meldungen ohne interne Details.

## 2. EU Cyber Resilience Act (CRA)

### 2.1 `POST /pastes` nicht implementiert — **kritisch**

**Befund:**  
`handler_create.go` enthält ausschließlich `writeError(w, http.StatusNotImplemented, "not implemented")`. Damit fehlen zentrale sicherheitsrelevante Anforderungen vollständig:

- AC-11: Body-Limit von 1 MiB (`http.MaxBytesReader`) fehlt.  
- AC-13: Validierung von `expires_in_seconds` (positiv, begrenzt auf max. 10 Jahre) fehlt.  
- AC-14: ID-Erzeugung mit `crypto/rand` und mind. 64 Bit Entropie fehlt.  
- AC-01 und AC-07: Anlegen und Validierung ungültiger Eingaben fehlen.

**Konkrete Abhilfe:**  
`handler_create.go` vollständig implementieren:
- `r.Body = http.MaxBytesReader(w, r.Body, 1<<20)` vor dem JSON-Decoding setzen.  
- Bei Überschreitung mit `413` antworten.  
- `content` als Pflichtfeld validieren, leeren Content mit `400` ablehnen.  
- `expires_in_seconds` als positive Ganzzahl validieren und auf max. 10 Jahre begrenzen.  
- ID mit `crypto/rand` erzeugen, z. B. 16 Hex-Zeichen (64 Bit Entropie) über `encoding/hex`.  
- Erfolgreiche Anlage mit `201` und JSON-Antwort inkl. `id` beantworten.

### 2.2 Fehlende Server-Timeouts / Härtung — **hoch**

**Befund:**  
`main.go` erzeugt einen `http.Server` ohne `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout` oder `MaxHeaderBytes`. Das ermöglicht Slowloris und Ressourcenerschöpfung und widerspricht dem Grundsatz „security by design/default“ des CRA.

**Konkrete Abhilfe:**  
In `main.go` im `http.Server` ergänzen:
```go
ReadHeaderTimeout: 5 * time.Second,
ReadTimeout:       10 * time.Second,
WriteTimeout:      10 * time.Second,
IdleTimeout:       60 * time.Second,
MaxHeaderBytes:    1 << 20,
```
(Import `time` ergänzen.)

### 2.3 Keine TLS-Terminierung im Code — **mittel**

**Befund:**  
`main.go` startet mit `server.ListenAndServe()` ohne TLS. Bei direkter Exposition im Netz wäre der Transport unverschlüsselt.

**Konkrete Abhilfe:**  
- Entweder `ListenAndServeTLS` mit Zertifikaten verwenden oder  
- im Deployment einen TLS-terminierenden Reverse-Proxy vorschalten und dies in `README.md` verbindlich dokumentieren.

### 2.4 Kein Rate-Limiting / Quota — **mittel**

**Befund:**  
Der Dienst ist als In-Memory-Pastebin öffentlich erreichbar, aber es gibt keine Begrenzung der Request-Anzahl oder der Anzahl gespeicherter Pastes. Ein Angreifer kann den Speicher durch massenhafte `POST`-Anfragen erschöpfen.

**Konkrete Abhilfe:**  
- In `routes()` eine einfache Rate-Limit-Middleware pro Client-IP einführen (z. B. Token-Bucket).  
- Zusätzlich eine maximale Anzahl gleichzeitig gespeicherter Pastes im `Store` definieren und bei Überschreitung `503` oder `429` liefern, ohne Produktfunktion unverhältnismäßig zu brechen.

### 2.5 Fehlender Security-Header — **niedrig**

**Befund:**  
`response.go` setzt nur `Content-Type`. Für JSON-Antworten fehlt `X-Content-Type-Options: nosniff`.

**Konkrete Abhilfe:**  
In `writeJSON` vor `WriteHeader` ergänzen:
```go
w.Header().Set("X-Content-Type-Options", "nosniff")
```

### 2.6 Fehlende SBOM / Sicherheitsdokumentation — **niedrig**

**Befund:**  
Es sind keine externen Abhängigkeiten sichtbar (nur Standardbibliothek), was gut ist. Jedoch ist im geprüften Code keine SBOM oder dokumentierte Sicherheits-/Update-Strategie erkennbar.

**Konkrete Abhilfe:**  
- In `README.md` einen Abschnitt „Security / SBOM“ aufnehmen: verwendete Go-Version, keine externen Dependencies, Support-/Patch-Prozess, gemeldete Schwachstellen.  
- `go.mod` als SBOM-Quelle nutzen und im Release dokumentieren.

### 2.7 Fehlende Tests für den Create-Handler — **mittel**

**Befund:**  
`handler_create_test.go` existiert nicht; nur `main_test.go` testet Health und 405. AC-09 (alle Handler inkl. Fehlerpfade) ist nicht erfüllt, insbesondere für den wichtigsten und sicherheitskritischsten Endpunkt.

**Konkrete Abhilfe:**  
`handler_create_test.go` ergänzen mit Tests für:
- Erfolgreiche Anlage (201, eindeutige IDs)  
- Ungültiges JSON (400)  
- Leerer Content (400)  
- `expires_in_seconds` negativ / zu groß (400)  
- Body > 1 MiB (413)  
- Prüfung, dass keine ID sequenziell vorhersagbar ist (nur stichprobenartig).

## 3. EU AI Act

Keine KI-Funktion im sichtbaren Code. Der AI Act ist nicht anwendbar.

## 4. Pflichttexte & UI

Kein öffentliches Web-UI vorhanden. Impressums-/Cookie-/Consent-Pflichten sind für ein reines Backend nicht einschlägig.

## 5. Accessibility

Kein öffentliches Web-UI vorhanden. WCAG/BITV/EAA nicht anwendbar.

---

**Zusammenfassung:**  
Der Hauptgrund für `CHANGES_REQUESTED` ist der nicht implementierte `POST /pastes`-Handler. Dadurch fehlen gleich mehrere sicherheitsrelevante Pflichten aus AC und CRA. Daneben sind Server-Härtung, Speicherbegrenzung und Rate-Limiting zu ergänzen. Ein fundamentaler DSGVO-Verstoß oder ein klares Leck personenbezogener Daten ist im aktuellen sichtbaren Stand nicht vorhanden, daher kein `BLOCKED`.