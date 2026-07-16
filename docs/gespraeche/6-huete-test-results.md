# 6-Hüte-Modell Test über 3 Provider

**Datum:** 2026-02-25
**Test:** Multi-Provider Verteilung via sigoREST

## Ergebnisse

| Hut | Provider | Modell | Antwort |
|-----|----------|--------|---------|
| ⚪ **WEISS** | mammouth.ai | claude-h | Go GC Fakten (Mark-and-Sweep, Stack vs Heap) |
| 🔴 **ROT** | moonshot.ai | kimi | "Token Bucket zuerst." |
| ⚫ **SCHWARZ** | z.ai | zai-glm46 | Performance-Frust, Paradigmen-Krampf, Ökosystem-Fiasko |
| 🟡 **GELB** | mammouth.ai | claude-s | Integration, Flexibilität, Erweiterbarkeit |
| 🟢 **GRÜN** | moonshot.ai | moon-8k | Concurrent Lisp, KI/ML, Cloud, Meta-Programmierung |
| 🔵 **BLAU** | z.ai | zai-glm45 | Ökonomisch/technisch/gesellschaftliche Synthese |

## Technische Details

### Verwendete Modelle
- **mammouth.ai:** Anthropic/Claude (claude-h, claude-s)
- **moonshot.ai:** Kimi/Moonshot (kimi, moon-8k)
- **z.ai:** GLM (zai-glm46, zai-glm45)

### Features getestet
- ✅ Multi-Host Distribution via Model-Shortcodes
- ✅ Rate-Limiting (2s Ticker + 500ms Circuit-Breaker)
- ✅ `(sleep ms)` Primitive
- ✅ `(parfunc ...)` parallele Ausführung

### Code-Beispiel

```lisp
; 6 Hüte über 3 Provider verteilt
(parfunc sechs-huete
  ; mammouth.ai
  (sigo "Fakten..." "claude-h")
  (sigo "Chancen..." "claude-s")
  
  ; moonshot.ai
  (sigo "Gefühl..." "kimi")
  (sigo "Ideen..." "moon-8k")
  
  ; z.ai
  (sigo "Risiken..." "zai-glm46")
  (sigo "Meta..." "zai-glm45"))
```

## Fazit

Die Multi-Provider-Verteilung funktioniert erfolgreich. Die Rate-Limiting-
Mechanismen verhindern Überlastung. Verschiedene Provider liefern 
unterschiedliche Perspektiven (Claude: ausführlich, Kimi: technisch/prägnant,
GLM: direkt).
