# Retrospektive: Memory Management & 6-Hüte-Feature

**Sprint:** Memory Management & Multi-Provider Support  
**Datum:** 2026-02-25  
**Teilnehmer:** Gerhard Quell, Claude Opus 4.6

---

## Was haben wir erreicht?

### Features implementiert
1. **Singleton Nil Cell** - Reduziert Allokationen für ()/nil/leere Listen
2. **`eq` Primitive** - Pointer-Gleichheit für Identity-Tests
3. **`memstats`** - Go Runtime Memory-Statistiken
4. **`sleep ms`** - Pausen zwischen KI-Calls
5. **Rate-Limiting** - 2s Ticker + 500ms Circuit-Breaker in sigo
6. **Multi-Host Parameter** - `(sigo "..." "model" "" "http://host:9080")`

### Validierung
- ✅ 6-Hüte-Modell erfolgreich über 3 Provider (mammouth/moonshot/zai)
- ✅ Alle Tests pass
- ✅ Dokumentation aktualisiert

---

## Was lief gut? (🟢 Gelb)

| Aspekt | Details |
|--------|---------|
| **Planung** | Klare Optionen (A/B/C) mit Empfehlung |
| **Iterativ** | Schnell vom einfachen Singleton zum komplexen Multi-Host |
| **Zusammenarbeit** | Gute Abstimmung zwischen Prompts und Implementierung |
| **Tests** | Jede Änderung sofort getestet |
| **Dokumentation** | CLAUDE.md immer aktuell gehalten |

**Highlight:** Das 6-Hüte-Ensemble hat verschiedene Provider-Persönlichkeiten sichtbar gemacht:
- Claude (mammouth): Ausführlich, strukturiert
- Kimi (moonshot): Technisch, prägnant  
- GLM (z.ai): Direkt, knapp

---

## Was lief schwierig? (⚫ Schwarz)

| Problem | Ursache | Lösung |
|---------|---------|--------|
| **DNS nicht erreichbar** | mammouth/moonshot/zai nur im lokalen DNS | IP-Adressen verwenden oder /etc/hosts |
| **Circuit Breaker** | Zu viele parallele Requests | Rate-Limiting + sequenzielle Fallbacks |
| **Modell-Fehler** | kimi-thinking, kimi-instruct nicht verfügbar | Andere Modelle (moon-8k) nutzen |
| **Timeout bei parfunc** | 6 parallel Calls zu viel für einen Server | Host-Parameter für Verteilung |

**Erkenntnis:** Rate-Limiting muss client-seitig sein - der Server hat keinen globalen Ticker pro Client.

---

## Was haben wir gelernt? (🔵 Blau)

### Technisch
1. **Go's `time.Tick`** ist ideal für globale Rate-Limiter
2. **Mutex + Zeitstempel** schützt vor zu schnellen Calls
3. **Optionaler Host-Parameter** erweitert Funktionalität ohne Breaking Changes

### Prozess
1. **Testen mit echten Calls** deckt Probleme auf, die Unit-Tests nicht finden
2. **Mehrere Provider** = Redundanz + unterschiedliche Perspektiven
3. **Retrospektive** festhalten, bevor man zum nächsten Feature springt

---

## Action Items

| # | Aufgabe | Priorität |
|---|---------|-----------|
| 1 | Host-Parameter in README.md dokumentieren | Niedrig |
| 2 | `parfunc` mit Retry-Logik erweitern | Mittel |
| 3 | Provider-Healthcheck via `(sigo-host-health)` | Niedrig |
| 4 | Mehr Tests für Edge-Cases (negative sleep, etc.) | Mittel |

---

## Zitate

> "Code = Daten + KI = sich selbst erweiterndes System"  
> — Gerhard & Claude, Februar 2026

> "Token Bucket zuerst."  
> — Kimi (ROT), auf die Frage nach Rate-Limiting

---

## Fazit

Das Feature ist produktionsreif. Die Multi-Provider-Unterstützung eröffnet 
neue Möglichkeiten für Ensemble-Methoden. Das Rate-Limiting schützt vor 
Überlastung, bleibt aber transparent für den Nutzer.

**Nächster Schritt:** GoLisp selbst-dokumentierend machen? `(describe 'sigo)`
