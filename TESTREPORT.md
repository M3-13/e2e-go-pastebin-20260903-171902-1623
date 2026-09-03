VERDICT: BUGS_FOUND

## Fehler

**Titel:** Test-Suite kompiliert nicht — doppelte Testfunktionsnamen  
**Symptom:** `go test ./...` schlägt fehl, weil mehrere Testfunktionen im selben Paket doppelt deklariert sind. Dadurch bricht der Test-Build ab, und keine einzige Prüfung der REST-API (Acceptance Criteria) kann ausgeführt werden.  
**Repro:** Im Projektverzeichnis `go test ./...` ausführen.  
**Evidence:**
```
FAIL	pastebin [build failed]
.\pastebin_behavior_test.go:187:6: TestGetUnknownIDReturns404 redeclared in this block
	.\handler_read_test.go:58:6: other declaration of TestGetUnknownIDReturns404
.\pastebin_behavior_test.go:201:6: TestListReturnsMetadataWithoutContent redeclared in this block
	.\handler_list_test.go:29:6: other declaration of TestListReturnsMetadataWithoutContent
.\pastebin_behavior_test.go:254:6: TestDeleteUnknownIDReturns404 redeclared in this block
	.\handler_delete_test.go:42:6: other declaration of TestDeleteUnknownIDReturns404
```
**Suspected file(s):** `pastebin_behavior_test.go` (nicht in der Dateiliste des Branches aufgeführt, aber laut Testlauf vorhanden) sowie `handler_read_test.go`, `handler_list_test.go`, `handler_delete_test.go`. Die drei genannten Testnamen kollidieren paarweise; die gemeinsame Ursache ist die zusätzliche Datei `pastebin_behavior_test.go`, die dieselben Namen erneut deklariert. Die Duplikate müssen umbenannt oder die Datei entfernt werden.  
**Severity:** high

## Bekannte offene Entscheidungen
- MR !11 — wurde bewusst für die Entscheidung des Architekten offengelassen; sein Code kann auf `main` vorhanden sein oder nicht. Wird nicht als Fehler gewertet.