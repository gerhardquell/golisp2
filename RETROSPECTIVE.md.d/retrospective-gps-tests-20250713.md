# Retrospective: GPS-Tests und CL-Compat-Erweiterungen

**Datum:** 13. Juli 2026  
**Autor:** Gerhard Quell & kimi-k2.7-code  
**Feature:** Charakterisierungstests für GPS-Spracherweiterungen, `defvar`, `setf`, `defstruct`-Kollisionsvermeidung  
**Commit:** `292f900`

---

## Was wurde gebaut?

### 1. Tests für GPS-Spracherweiterungen (`pn-gps1/gps-tests.lisp`)
- 16 Charakterisierungstests für die durch den Norvig-GPS-Port gezogenen Spracherweiterungen
- Abdeckung: `union`, `set-difference`, `find-all`, `defstruct`, `setf`, `defvar`
- Lektion aus `pn-gps1/TODO.md`: *„Grün heißt: der Pfad ist grün“* — ein Feature, das auf einem Anwendungspfad läuft, braucht trotzdem eigene Tests.

### 2. `defvar` idempotent
- Zweites `defvar` für dasselbe gebundene Symbol ist nun no-op (Common-Lisp-Semantik).
- Benötigte neue Spezialform `bound?`, weil Primitive keinen Zugriff auf das aktuelle Environment haben.

### 3. `setf` für Accessor-Places
- `setf` erkennt registrierte Accessor-Places wie `(pt-x p)`.
- `defstruct` registriert Setter-Funktionen in `*setf-expanders*`.
- Da golisp2-Cells immutable sind, wird `(setf (pt-x p) 9)` zu `(set! p (set-pt-x p 9))` expandiert.

### 4. `defstruct`-Kollisionsvermeidung
- `(defstruct set (difference nil))` überschreibt `set-difference` nicht mehr stillschweigend.
- Bei belegtem Primärnamen wird die Alternative mit doppeltem Bindestrich verwendet (`set--difference`).

---

## Was lief gut?

### ✅ Testgetriebene Dokumentation
- Die Tests aus `TODO.md` haben sofort zwei echte Lücken sichtbar gemacht: `defvar`-Idempotenz und `setf`-Accessor-Places.
- Failing Tests vor dem Fix sind die objektive Wahrheit — keine Meinungsfrage.

### ✅ Spezialform statt Primitive
- `bound?` als Spezialform war der richtige Ort, weil Primitive kein Environment erhalten.
- Dies spiegelt die Architektur-Regel wider: *Braucht die Funktion `env`? → Spezialform.*

### ✅ Kollisionsvermeidung ohne Breaking Change
- Bestehende Structs wie `op` behalten ihre Accessor-Namen (`op-action`, `op-preconds` etc.).
- Nur bei tatsächlicher Kollision greift die Alternative.

### ✅ Wiederverwendung vorhandener Mechanismen
- `set-nth` als reine Lisp-Funktion für immutable List-Update.
- `*setf-expanders*`-Registry für generische `setf`-Erweiterbarkeit.

---

## Was war herausfordernd?

### ⚠️ Makros können das aktuelle Environment nicht sehen

**Problem:** `defstruct` ist ein Makro. Makros erhalten bei der Expansion nicht das aktuelle Evaluierungs-Environment, können also nicht direkt prüfen, ob ein Name bereits gebunden ist.

**Lösung:** Die Kollisionsprüfung mit `bound?` wurde in den zur Laufzeit ausgeführten Code verlagert. Accessor- und Setter-Funktionen werden dynamisch über `eval` definiert.

**Lesson Learned:** Wenn ein Makro Laufzeit-Informationen aus dem Environment braucht, muss der Check ins expandierte Code-Gerüst, oder das Feature wird zur Spezialform.

### ⚠️ Immutability vs. `setf`-Places

**Problem:** `(setf (pt-x p) 9)` sollte in CL den Slot des Objekts ändern. golisp2-Cells sind immutable — es gibt kein `rplaca`/`rplacd`.

**Lösung:** `setf` expandiert zu einer Variablen-Neuzuweisung: `(set! p (set-pt-x p 9))`. Das funktioniert für Variable-Places, nicht aber für komplexe Places wie `(setf (pt-x (car lst)) 9)`.

**Lesson Learned:** Immutable Datenstrukturen erfordern eine bewusste Design-Entscheidung bei `setf`. Die aktuelle Lösung ist pragmatisch, aber nicht vollständig CL-kompatibel.

### ⚠️ Stille Überschreibungen im globalen Env

**Problem:** `defstruct` hat existierende Funktionen wie `set-difference` überschrieben, ohne Warnung. Genau das warnt `CLAUDE.md` vor.

**Lösung:** Kollisionserkennung zur Laufzeit + alternative Namensgebung.

**Lesson Learned:** Das globale Env kennt kein `undefine`. Jede Definition, die andere Namen generiert, muss Kollisionsregeln haben.

---

## Offene Punkte

1. **Generische Places für `setf`:** Aktuell nur Accessor-Places mit Symbol-Argument (`(pt-x p)`). Komplexe Places wie `(car lst)` oder `(gethash key tbl)` werden nicht unterstützt.
2. **Kollisionsnamen vorhersagen:** `set--difference` ist funktional, aber nicht intuitiv. Eine dokumentierte Konvention oder ein expliziter `:accessor`-Parameter wären langfristig besser.
3. **Makro-Utils `defstruct`:** `tests/macro-utils.lisp` enthält ein eigenes `defstruct`-Makro mit einem defekten `symbol->string`. Das ist außerhalb des heutigen Scopes, aber potenziell verwirrend.

---

## Nächste Schritte (Vorschläge)

- [ ] `setf` für `car`/`cdr`-Places ergänzen
- [ ] `defstruct` optional mit explizitem Accessor-Namen ermöglichen
- [ ] `tests/macro-utils.lisp` bereinigen oder als veraltet markieren
- [ ] GPS-Tests in die reguläre Test-Suite (`./build/golisp2 -t` oder `go test`) integrieren
