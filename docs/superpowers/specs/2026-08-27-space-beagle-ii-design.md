# Space Beagle II — Design-Spec

**Datum:** 2026-08-27
**Status:** Entwurf, zur Review durch Gerhard
**Arbeitstitel:** Space Beagle II (Homage an A.E. van Vogt, „Die Weltraumexpedition" — und damit doppelt: an Darwins HMS Beagle)
**Projekt-Ort:** eigenes Repo `space-beagle-ii/` (Spiel) + `src/lib/fsrs.go` im golisp2-Repo (Domäne)

---

## Vision

Ein Lernspiel, das in einem Raumschiff spielt. Der Spieler steigt als Lehrling
in die Crew der Space Beagle II ein, wählt Fachrollen (Programmierer,
Bordingenieur, Xenobiologe, Astrophysiker) und wird zum Kommandanten ernannt,
wenn er alle Rollen erfolgreich absolviert hat. Gelernt wird ein Hybrid aus
**Spaced Repetition (FSRS)** für Wissen aller Art und **echtem Coden am
Schiffsterminal** — Lisp, Go, Python, C, JS laufen wirklich, lokal auf dem
Linux-System des Spielers. Zielgruppe: der Autor selbst (C), aber von Anfang
an veröffentlichungsfähig gebaut.

Die Lücke im Markt: Niemand deckt Multi-Sprachen-Programmieren (inkl. Lisp!)
plus Wissenschaften plus Browser in einem Spiel ab. CodeCombat (Programmieren,
2 Sprachen) und Brilliant (Wissenschaften, kein Coden) existieren; die
Schnittmenge ist leer.

**Das Alleinstellungsmerkmal:** Spielinhalt ist homoikonische Lisp-Daten.
Räume, Crew, Quests, Ereignisse — alles `define-*`-Datenblätter, heiß
nachladbar, KI-generierbar über sigo, maschinell verifizierbar über exec.
Das Spiel erweitert sich mit genau dem, was es lehrt.

---

## Entscheidungshistorie (Brainstorming 2026-08-27)

| Entscheidung | Gewählt |
|---|---|
| Zielpublikum | C: für den Autor, aber veröffentlichungsfähig |
| Kern-Schleife | Hybrid: FSRS-Kern + Boss-Level mit echtem Code |
| Ausführung | lokal auf dem Linux-System (exec); Smartphone explicit KEIN Ziel v1 |
| Setting | Raumschiff, Räume = Fächer, Crew = Mentoren |
| Technik Client | Browser (Canvas 2D + DOM-Schicht), serviert über golisp2 `webserv` |
| Grafik | Pixel-Art retro, CC0-Assets (Kenney etc.), limitierte Palette |
| Perspektive | Top-Down-Deckplan; Computerkern als Side-View (inszenierter Lern-Ort) |
| Rollensystem | Lehrling → 4 Fachrollen → Kommandant; Bord-KI nicht wählbar (Tutorin) |
| Content-Pipeline | A+B: handkuratiertes Gerüst + KI-generierte Masse (sigo) |
| Wiederholung | FSRS (open-spaced-repetition/go-fsrs), diegetisch als Schiffswartung |
| Fortschritt | Kompetenz = FSRS-Stabilität + gelöste Bosse, nicht XP |

---

## 1. Gesamtarchitektur

```
┌──────────────────────── Browser (Client) ────────────────────────┐
│  Canvas 2D: Pixel-Schiff, Top-Down-Deckplan, Crew-Sprites        │
│  DOM-Schicht: Dialoge (Crew/Bord-KI), Inventar, Quest-Log        │
│  CodeMirror: Code-Terminal im Computerkern (Side-View)           │
│  JS-Terminals: Ausführung im Browser des Spielers (immer)        │
│  WebAudio: Chiptune-Sounds, Reaktor-Brummen                      │
│  boot.js (besteht schon in golisp2): golisp.call / golisp.on     │
└───────────────△ WebSocket (ws-export/ws-emit) ────────────────────┘
┌───────────────▽ golisp2-Server (Linux des Spielers) ─────────────┐
│  Spielzustand: Spieler, Rollen, Fortschritt — Lisp-Hash-Tables   │
│  Persistenz: JSON-Datei (file-write/file-read)                   │
│  FSRS: src/lib/fsrs.go (go-fsrs, MIT) — neue golisp2-Domäne      │
│  Quest-Engine + Weltmodell: alles Lisp-Daten (s. Abschnitt 8)    │
│  Code-Ausführung: exec (gcc/go/python3/node/golisp2)             │
│    pro Sprache ein Terminal-Profil: Kommando, Zeitlimit, tmp/    │
│  Bord-KI: regelbasiert (v1) — LLM-Hook (sigo) optional später    │
└───────────────────────────────────────────────────────────────────┘
```

Trennung: Der Client weiß nichts — er zeigt, was der Server sagt. Server ist
die einzige Quelle für Spielzustand, Quest-Bewertung, FSRS-Termine. Basis ist
die existierende webserv/WebSocket-Architektur von golisp2.

Rendering-Konzept: kleines Grundraster (z. B. 320×180), hochskaliert mit
`image-rendering: pixelated` (Integer-Skalierung, scharf auf jedem Monitor).
16×16- oder 32×32-Tiles, Tilemaps aus Tiled (JSON-Export), Sprite-Sheets als
PNG. Limitierte Palette (16–32 Farben) für kohärenten Look über gemischte
Asset-Quellen. Font: Pixel-Font nur für Headlines (VT323/Silkscreen); der
Code-Editor nutzt eine normale Mono-Font — Lesbarkeit schlägt Stil.

---

## 2. Rollen, Ränge & Progression

Kompetenz wird nicht in XP gemessen, sondern in **FSRS-Stabilität + gelösten
Boss-Leveln**. Rangaufstieg erfordert nachweislich gefestigtes Wissen —
Gedächtnis als Spielmechanik.

```lisp
(define-role 'programmierer
  :raum    'computerkern
  :rangfolge '(terminal-kadett systemoffizier chefprogrammierer)
  :linien  '((lisp . questlinie-lisp) (python . questlinie-python)
             (js . questlinie-js) (go . questlinie-go) (c . questlinie-c))
  :kernkarten '(car cdr lambda rekursion closure pointer slice goroutine)
  :boss     'boss-schiffswaende-reparieren)
```

| Rolle | Raum | Curriculum | Abschluss |
|---|---|---|---|
| Lehrling (Start) | alle | Schiffskunde, Terminal-Basics, Kostprobe je Fach | alle Kostproben → Fachrollen frei |
| Programmierer | Computerkern | 5 Sprachen-Questlinien (Reihenfolge frei) | alle Linien + Boss |
| Bordingenieur | Maschinenraum | Elektrotechnik, Materialkunde, Antriebssysteme | Linien + Boss |
| Xenobiologe | Labor | Bio, Chemie, Alien-Ökologie | Linien + Boss |
| Astrophysiker | Brücke | Mechanik, Optik, Astronomie, Orbitaldynamik | Linien + Boss |
| Kommandant (Meta) | Brücke | Kommandos-Quests (Planung, Architektur, Systemdenken) | alle 4 Fachrollen abgeschlossen |
| Bord-KI | überall | nicht wählbar: Tutorin, Questgeberin, FSRS-Ansagerin | — |

Drei Rangstufen pro Rolle. Aufstieg: Quest-Linien der Stufe gelöst **und**
zugehörige Kernkarten mit FSRS-Stabilität ≥ X Tagen. Boss-Level =
Aufstiegsprüfung am Terminal (mehrere Versuche erlaubt; Versuchszahl fließt
ins FSRS-Rating).

FSRS-Bibliothek gilt rollenübergreifend: gefestigtes Wissen bleibt beim
Rollenwechsel erhalten — die Rolle ist der Pfad, das Wissen der Charakter.

Persistenz: ein Spielerprofil als JSON (Name, Rolle, Rang, gelöste Quests,
FSRS-Zustände pro Karte, Logbuch). Datenmodell von Anfang an
mehrspielerfähig (Map Spieler → Profil), v1 bleibt Single-Player.

---

## 3. Quest-Formate & die Lern-Schleife

Drei Quest-Typen in v1:

1. **Wissens-Quest (Quiz)** — Multiple-Choice, Freitext, Zuordnung. FSRS-Karte pur.
2. **Code-Lückentext** — echtes Code-Fragment mit Lücke(n), typbar oder Choice.
3. **Terminal-Quest (Boss-Level)** — echter Code am Computerkern-Terminal, Ausführung über `exec`, Ausgabe wird geprüft.

Quest = einheitliches Datenblatt:

```lisp
(define-quest 'lisp-car-01
  :karte 'car :linie 'lisp :typ 'quiz
  :frage "Was liefert (car '(1 2 3))?"
  :antworten ((t "1") () "(1 2 3)" () "(1)")
  :hinweis "car nimmt das ERSTE Element — nicht die Liste selbst."
  :erklaerung "car = Contents of Address Register, historisch: ...")

(define-quest 'python-for-01
  :karte 'python-for :linie 'python :typ 'lueckentext
  :code "for i in ____ (10):\n    print(i)"
  :loesung '("range") :sprache 'python)

(define-quest 'boss-rumpfscan
  :karte 'go-slices :linie 'go :typ 'terminal
  :aufgabe "Der Rumpf zeigt Mikrorisse. Schreibe ein Programm, das
            sensordaten.txt scannt und alle Sektionen mit >3 Anomalien meldet."
  :sprache 'go
  :startdatei "rumpf.go"
  :pruefung '("go run rumpf.go sensordaten.txt")
  :erwartet "ALARM: Sektion 4\nALARM: Sektion 9"
  :lueckentext-fallback boss-rumpfscan-fallback)   ; für --published
```

**Karte = Wissensatom** (eine Karte prüft genau eine Sache). Mehrere Quests
können dieselbe Karte füttern — Quiz, Lückentext und Boss zu `closure` sind
drei Türen zu einer Erinnerung. Der Spielzustand wächst mit den Karten.

Lösung → FSRS-Rating: falsch/nicht gelöst = 1 · richtig nach Hinweis / Boss
mit >3 Versuchen = 2 · sauber / Boss in 2–3 Versuchen = 3 · sofort, flüssig /
Boss im 1. Versuch = 4. Falsch beantwortete Quests werden sofort aufgeklärt
(`:erklaerung`) — FSRS plant die Wiederholung, das Spiel erklärt jetzt.

Kern-Schleife: Bord-KI funkt (fällige Protokolle + neue Mission) → Spieler
läuft in den Raum → Quest durch Crew/Bord-KI → lösen → Server bewertet,
FSRS-Karte fortschreiben → System stabilisiert sich sichtbar → zurück zum
Deckplan.

Content-Organisation: `content/`-Verzeichnis, eine Lisp-Datei pro Linie,
geladen beim Start, heiß nachladbar. Neue Fächer = neue Datei, kein Code.

---

## 4. FSRS-Diegese — Schiffssysteme statt Karteikarten

Man wiederholt keine Karten — **man wartet Schiffssysteme.** Jede FSRS-Karte
ist ein System an Bord, das einmal in Betrieb genommen wurde und instandzu-
halten ist. Die Verknüpfung läuft über die Quest-Linie (also den Raum).

Retrievability (R, 0…1) steuert die Anzeige:

```
R ≥ 0.9   grün — summt zufrieden        R ≥ 0.7   gelb — Wartung empfohlen
R < 0.7   orange/flackernd — Protokoll aktiv
R < 0.5   rot — das System braucht dich dringend
```

Raum-Stabilität = Mittelwert des R über seine Raum-Karten (Display-Wert;
die FSRS-Planung bleibt pro Karte, nie pro Raum). Der Computerkern
flackert, wenn das Lisp verblasst — man sieht das eigene Vergessen als
Schiffszustand.

Daily Loop beim Boarding: Bord-KI-Einsatzplanung („3 Wartungsprotokolle
fällig, neue Mission verfügbar; Empfehlung: erst Wartung"). Fällige Reviews
laufen als Serie am flackernden Gerät; richtige Antwort = System beruhigt
sich hörbar (Sound), 3 in Folge = Combo-Anzeige.

**Kein Bestrafungs-Design:** Rote Systeme bedeuten nie Schaden, keine
Streak-Shows. Rot ist eine Einladung; FSRS plant die Karte ohnehin wieder
ein. Konsequenzen trägt nur die Mechanik (Rangaufstieg braucht Stabilität).

Zeit ist echte Zeit: FSRS rechnet mit Kalendertagen; das Spiel belohnt
Wiederkommen, nicht Marathon-Sessions. Kommandanten-Kriterium: alle Rollen
abgeschlossen und alle Rollenräume dauerhaft auf hohem Stabilitäts-Niveau.

---

## 5. Content-Pipeline — Hand-Gerüst + KI-Masse

| Handkuratiert (A) | KI-generiert (B, sigo) |
|---|---|
| Rollen, Räume, Quest-Linien-Skelett | Fragen-Pools: 3–5 Quiz-Varianten pro Karte |
| Kernkarten-Definitionen | Lückentexte zu bestehenden Karten |
| Boss-Level (Aufgabe, Sensordaten, Tests) | Boss-Varianten (weitere Datensätze) |
| Crew-Persönlichkeiten, Bord-KI-Stimme | Übersetzungen (i18n, später) |
| Onboarding / Lehrlings-Quests | |

Das Gerüst gibt die Wahrheit vor, die KI liefert Variation. Generator ist
selbst ein Terminal-Befehl am Computerkern (diegetisch: der Computerkern
schreibt seine eigenen Trainingsmodule):

```lisp
(gen-quests 'car :n 4 :model "claude-h")
; → sigo-Prompt aus Karten-Definition + vorhandenen Quests
; → Vorschlag nach content/vorschlaege/car-neu.lisp
; → automatische Gültigkeitsprüfung (Pflichtfelder, Karte existiert,
;   Antworttypen, keine Duplikate — der redefine-Guard für Content)

(gen-quests 'alles :model-ensemble '("claude-h" "gemini-p" "gpt41"))
; → parfunc: drei Modelle parallel, Auswahl des Besten
```

Vorschläge landen nie direkt im Spiel: `content/vorschlaege/` → menschliche
Revision → Übernahme in `content/<linie>.lisp` → Load. Centaur-Prinzip als
Content-Schleife.

Maschinelle Verifikation, wo möglich: Terminal-Quests werden live gegen den
lokalen Executor getestet (generierter Boss mit `:pruefung`/`:erwartet`
muss wirklich bestehen, bevor er spielbar wird). Quiz/Lückentext:
Ensemble-Kreuzcheck per parfunc (Widerspruch zwischen Modellen = Flagge),
endgültig menschliche Revision bei Übernahme.

Startzustand: ~50 handkuratierte Karten, ~150 Quests (Lehrling + Lisp-Linie
vollständig — Dogfooding). Danach Wachstum per `gen-quests` bei Bedarf.

---

## 6. Betriebs- & Sicherheitsmodell

Grundkonflikt: Terminal-Quests führen echten Code aus. Solo unkritisch
(eigener Code, eigener Rechner); veröffentlicht niemals fremder Code über
`exec` auf dem Host.

Drei Betriebsmodi, eine Binary:

| Modus | Start | Terminals | Bord-KI |
|---|---|---|---|
| Solo (v1) | `golisp2 spiel.lisp` | alle 5 Sprachen via exec lokal | regelbasiert, LLM-Hook frei |
| Veröffentlicht | `--published` | nur Browser-Terminals | regelbasiert, LLM aus/rate-limited |
| Self-Host (später) | wie Solo | alles — deren Rechner | deren sigo-Keys |

Schlüssel: **JS-Terminals laufen immer im Browser des Spielers** (sandboxed,
meldet nur das Ergebnis). Python im veröffentlichten Modus per Pyodide (WASM,
im Spieler-Browser). Degradations-Matrix:

| Sprache | Solo | Veröffentlicht |
|---|---|---|
| JS | Browser | Browser — identisch |
| Python | exec python3 | Pyodide im Browser |
| Go | exec go run | Lückentext-Fallback |
| C | exec gcc | Lückentext-Fallback |
| Lisp | exec ./golisp2 | Lückentext-Fallback |

Jeder Boss trägt optional `:lueckentext-fallback` — dieselbe Karte, geringere
Könnens-Tiefe. Der volle Computerkern ist ein Self-Host-Feature (ehrlich
kommuniziert; für die Zielgruppe eine Lektion, kein Mangel).

Lokale Ausführung gezähmt aus Höflichkeit, nicht Paranoia: Terminal-Quests
in `./tmp/spiel/`, Zeitlimit (Endlosschleife beendet sich selbst),
Ausgabe-Deckel, Aufräumen danach.

Spielerprofile: Solo = eine JSON-Datei. Veröffentlicht = Namenszugang (Name →
Profil), kein Account-System in v1. Datenmodell trägt später alles nach.

---

## 7. Meilensteine, Testing, Non-Goals

| M | Inhalt | Abnahme-Beweis |
|---|--------|----------------|
| M1 — Skelett | webserv + Canvas, Deckplan (Tiled-JSON), Spieler-Sprite, Raumwechsel, DOM-Dialoge, Bord-KI regelbasiert, Spielstand-JSON | Durchs Schiff laufen |
| M2 — Quiz-Kern | src/lib/fsrs.go, define-quest, Quiz-Quests, Rating-Mapping, Wartungsprotokolle, Systemflackern nach R | Erste echte Lern-Session (Lehrling + Lisp-Linie) |
| M3 — Computerkern | Side-View-Raum, CodeMirror-Terminal, Terminal-Profile via exec, Lückentexte, erster Boss | Echter Go-Boss am Terminal gelöst |
| M4 — Rollen | Ränge, 4 Fachrollen + Content, Kommandant, Crew-Porträts, Sound | Aufstieg Lehrling → Kommandant möglich |
| M5 — Veröffentlichung | --published, JS-Browser-Terminal, Pyodide, Fallbacks, Namenszugang | Fremde spielen ohne Host-exec |
| M6+ — später | gen-quests, Bord-KI-LLM, Ensemble-Kreuzcheck, i18n, Kommandos-Quests | — |

M1+M2 bewusst klein: nach M2 ist es ein Lernspiel, alles danach ist Ausbau.

Testing (golisp2-Prinzipien): Go-Unit-Tests für FSRS-Wrapper, Rating-Mapping,
Quest-Validierung; Lisp-Testsuite (-t-Muster); Content-Validierung beim Load
(ungültige Quest = Startup-Fehler, nie Laufzeit-Überraschung); Terminal-
Profile im Test gegen echte Compiler; E2E nach tests/golisp2web-test.lisp-
Muster (parfunc + http-wait). Wichtigster Test: der Autor spielt.

Non-Goals v1: Multiplayer/Echtzeit, Accounts/Passwörter, 3D/Physik,
Action-Kampf, native Mobile-App, eigene Pixel-Art von Hand, LLM-Bord-KI.

---

## 8. Welt-Erweiterbarkeit — Ereignisse, Personen, Orte

Ausdrückliches Design-Ziel: die Spielwelt ist offen für Erweiterung in alle
Richtungen. Nicht nur Quests — **jede Entität ist ein Lisp-Datenblatt** mit
demselben Muster:

```lisp
(define-raum 'labor
  :tilemap "assets/tilemaps/labor.json"
  :farbfeld 'labor-stimmung          ; Licht/Atmosphäre
  :systeme '(stoffwechselfilter zentrifuge probenarchiv))

(define-crew 'dr-vesna-xenobiologin
  :raum 'labor :rolle 'xenobiologe
  :portrait "assets/crew/vesna.png"
  :persoenlichkeit '(neugierig praezise ungeduldig-mit-schlamperei)
  :begruessung "Neue Probe? Her damit. Und diesmal beschriftet, bitte.")

(define-ereignis 'coeurl-kontakt        ; Van-Vogt-Hommage
  :ausloeser '(und (= rang 'chefprogrammierer) (> tag 30))
  :phasen '(anomalie-entdeckt (bio+technik+physik . questkette)
           (entscheidung . drei-ausgaenge))
  :folgen '((erfolg . freischaltung-messraum)
            (misserfolg . rumpfschaden-ereigniskette)))
```

- **Orte:** neue Räume/Decks per define-raum + Tilemap-Datei — der Deckplan
  ist eine Liste, kein hartes Level. Das Schiff kann wachsen (Erweiterungs-
  module andocken!).
- **Personen:** Crew per define-crew, mit Persönlichkeit, die Dialoge und
  Quest-Vergaben prägt. Neue Crew-Mitglieder = neue Quest-Linien.
- **Ereignisse:** Missionsbögen per define-ereignis mit Auslösern (Rang, Tag,
  Fortschritt), Phasen und Folgen — mehrstufige Geschichte über Fächer hinweg
  (eine Anomalie braucht Bio UND Technik UND Programmieren: crossover-Lernen).
  Die Coeurl-Kette ist Referenz und Namens-Hommage.
- Alles heiß nachladbar, alles KI-generierbar über dieselbe Pipeline wie
  Quests (Vorschlag → Prüfung → Revision → Übernahme), alles Versionierbar
  in content/-Dateien.

Damit ist die Welt kein festes Level, sondern ein wachsendes Daten-
Universum — die Erweiterbarkeit ist kein Feature, sie ist das Material.

---

## Offene Fragen (für Implementierungsplanung)

1. **Rang-Schwellwerte:** FSRS-Stabilität X für Rangaufstieg — konkrete
   Tageswerte je Rolle (Faustregel: 7/21/60 Tage für Rang 1/2/3?).
2. **FSRS-Modul-API:** exakte Primitive-Signatur in golisp2 (fsrs-review
   State vs. State im Spielzustand halten) — beim M2-Plan festlegen.
3. **Client-Framework-Bindung:** plain Canvas + eigenem Mini-Loop vs.
   PixiJS — Entscheidung in M1 (starte plain, wechsle bei Bedarf).
4. **Pyodide-Integration (M5):** Bundle-Größe und Offline-Fähigkeit klären.
5. **Lizenz des Spiels:** golisp2 ist Unlicense; Spiel analog? Assets CC0
   sind vereinbar.

---

*Spec aus dem Brainstorming 2026-08-27 (Gerhard Quell & Claude/GLM-5.3).
Sieben Abschnitte plus Erweiterbarkeit, Abschnitt für Abschnitt im Dialog
abgenickt.*
