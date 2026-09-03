# TODO - Aufgabenplanung 20260903-1

/brainstorming

## Ergänzungen von golisp2 um: 
- [x] mathematische Funktionen 
    - truncate  (CL: 2 Werte, Richtung Null)
    - ceil      (als `ceiling`, CL-Name analog `floor`; 2 Werte)
    - round     (CL: Rundung-zum-Geraden bei .5; 2 Werte)
    - parse-Int    => aus string Integer  (als `parse-int`)
    - parse-Float  => aus string Float    (als `parse-float`)
- [x] Einbindung maxima als Rechensystem
    können wir maxima entweder als externes programm via exec
    nutzen und uns dazu eine Art programmierbaren Taschenrechner
    bauen, der maxima nutzt.
    → `src/lib/maxima.go`: `maxima-open`/`maxima-eval`/`maxima-close`,
      Long-Lived-Prozess (Zustand bleibt), Sentinel-Sync statt
      Prompt-Parsing, Timeout-Schutz gegen blockierende Rückfragen
      (z.B. `integrate(x^n,x)` → "is n equal to -1?"). Ergebnis bisher
      als roher String (Maxima-Linearsyntax) — keine Parsing-Schicht
      nach golisp2-Datenstrukturen (siehe Diskussion: `:lisp`-Bridge
      wäre nächster Ausbau für strukturierte Resultate).
 



