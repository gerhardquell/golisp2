# GoLisp2 – Memory Management

GoLisp2 vertraut vollständig auf Go's Garbage Collector. Es gibt **kein**
manuelles Memory-Management.

## Wie es funktioniert

- **Cell-Allokation:** jedes `&Cell{}` landet auf dem Go-Heap
- **Kein Object-Pooling:** keine `sync.Pool` o. ä.
- **Zirkuläre Referenzen:** Go's GC erkennt Zyklen (Lambdas, `labels`)
- **Singleton-Nil:** `MakeNil()` gibt immer dieselbe Instanz zurück
- **Small-Int-Cache:** kleine Ganzzahlen werden wiederverwendet (`src/lib/types.go`)

## Singleton-Nil

Vorher erzeugte jedes `()`, `nil`, jede leere Liste eine neue Cell.
Jetzt teilen sich alle dieselbe `nilCell`-Instanz.

```lisp
(eq (list) (list))   ; → t  (identische Pointer)
(eq nil nil)         ; → t
(eq '() '())         ; → t  (auch quote-nil ist identisch)
```

**Thread-Sicherheit:** sicher für `parfunc` — die Singleton-Nil wird nur
gelesen, nie modifiziert. Wer das ändert, bricht die Nebenläufigkeit.

Konsequenz für Vergleiche: siehe `docs/lisp-semantik.md` (`eq` vs. `equal?`).

## `(memstats)`

Gibt die aktuellen Go-Runtime-Stats als Assoc-Liste zurück:

```lisp
(memstats)
;; => ((heapalloc    . 421376)    ; aktueller Heap in Bytes
;;     (heapsys      . 7864320)   ; vom OS reservierter Heap
;;     (heapobjects  . 1247)      ; Anzahl allozierter Objekte
;;     (numgc        . 5)         ; Anzahl GC-Zyklen
;;     (pausetotalns . 234567)    ; totale GC-Pause in Nanosekunden
;;     (totalalloc   . 1234567))  ; kumulative Allokation
```

## Best Practices

1. **Keine Angst vor Allokationen.** Go's GC ist für kurzlebige Objekte
   optimiert. Nicht spekulativ optimieren — erst messen (`perfTodo.md`).
2. **Externe Ressourcen explizit schließen:** PostgreSQL-Verbindungen mit
   `pg-close`, Shared-Memory mit `shm-detach` / `shm-remove`.
3. **Das globale Environment wächst permanent** — es gibt kein `undefine`.
   Bei langlaufenden Prozessen relevant.
4. **Monitoring:** bei Langzeit-Prozessen regelmäßig `(memstats)` loggen.
