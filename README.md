# Pastebin-REST-API

Eine kleine, eigenständige Pastebin-REST-API in Go. Pastes werden thread-sicher im
Arbeitsspeicher gehalten (optional mit Ablaufzeit) und über vier JSON-Endpunkte
verwaltet: anlegen, abrufen, auflisten und löschen. Es werden ausschließlich
Standardbibliotheks-Pakete (`net/http`, `encoding/json`, `sync`, `crypto/rand`, …)
verwendet — keine externen Abhängigkeiten und keine externen Dienste.

## Tech-Stack

- **Sprache**: Go (1.22+)
- **Framework**: `net/http` (Standardbibliothek, ServeMux mit Pfadmustern)
- **Modul**: `pastebin` (`go.mod`)
- **Tests**: `go test` mit `httptest`
- **Speicher**: In-Memory mit `sync.RWMutex`

## Installation & Start

```sh
go build -o pastebin-api .
./pastebin-api
```

Oder direkt aus dem Quellcode:

```sh
go run .
```

Der Server startet auf Port 8080.

## Tests

```sh
go test ./...
```

## API-Endpunkte

| Methode | Pfad           | Beschreibung                                        |
|---------|----------------|-----------------------------------------------------|
| GET     | `/healthz`     | Health-Check, antwortet `200 {"status":"ok"}`       |
| POST    | `/pastes`      | Paste anlegen, antwortet `201 {"id":"..."}`         |
| GET     | `/pastes/{id}` | Einzelne Paste abrufen, antwortet `200` mit Inhalt  |
| GET     | `/pastes`      | Metadaten aller Pastes auflisten, `200` JSON-Array  |
| DELETE  | `/pastes/{id}` | Paste löschen, antwortet `204` ohne Body            |

Nicht unterstützte HTTP-Methoden auf `/pastes` oder `/pastes/{id}` werden mit
`405 {"error":"..."}` beantwortet.

### Beispiel

```sh
# Paste anlegen
curl -X POST http://localhost:8080/pastes \
  -H "Content-Type: application/json" \
  -d '{"content":"hello world","language":"text"}'

# Paste abrufen
curl http://localhost:8080/pastes/<id>

# Metadaten auflisten
curl http://localhost:8080/pastes

# Paste löschen
curl -X DELETE http://localhost:8080/pastes/<id>
```
