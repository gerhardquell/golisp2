# GoLisp auf ModelScope – Planungsdokument

**Status:** Überlegungsphase
**Datum:** 2026-02-27
**Auslöser:** Chinesische README ist auf GitHub, jetzt Überlegung für ModelScope-Präsenz

---

## Kontext

- GoLisp hat jetzt eine vollständige chinesische README (`README_CN.md`)
- Ziel: Präsenz auf modelscope.cn (chinesischer KI-Hub, ähnlich HuggingFace)
- Eigene Infrastruktur verfügbar: Debian-Server mit nginx, root-Zugang, sigoREST-Server

---

## Optionen im Überblick

### Option A: ModelScope Space (Standalone)
**Beschreibung:** Komplette Demo auf ModelScope-Infrastruktur
**Technik:** Gradio + Docker, GoLisp-Binary im Container

**Vorteile:**
- Keine eigene Server-Administration
- Natürliche Integration in ModelScope-Ökosystem
- Einfache Discovery durch chinesische Community

**Nachteile:**
- Cold-Start-Problematik (Server schläft bei Inaktivität)
- Resource-Limits (CPU/RAM/Timeout)
- KI-Kosten nicht kontrollierbar bei offenem Zugang

**KI-Kosten-Schutz:**
- Demo-Modus mit Mock-Antworten
- BYOK (Nutzer müssen eigenen Key/Server angeben)
- Statische Code-Beispiele statt Live-Ausführung

---

### Option B: Eigenständiger Server
**Beschreibung:** GoLisp-WebREPL läuft auf eigenem 24/7 Server
**Technik:** nginx reverse proxy → GoLisp-Service (WebSocket oder HTTP)

**Vorteile:**
- Volle Kontrolle über Ressourcen
- Keine Cold-Starts
- Eigene Rate-Limiting/Authentifizierung möglich

**Nachteile:**
- Höherer Admin-Aufwand
- Keine native ModelScope-Integration (nur als externer Link)

**KI-Kosten-Schutz möglich durch:**
- nginx Basic Auth (`.htpasswd`)
- IP-Whitelist (`allow/deny`)
- sigoREST-seitiges Rate-Limiting
- Session-basierte Limits (max X KI-Aufrufe pro Session)

---

### Option C: Hybrid (Empfohlene Richtung)
**Beschreibung:** ModelScope Space als "Showcase", aber KI-Calls gehen an kontrollierten eigenen Server
**Technik:**
- Minimaler Space auf ModelScope (nur UI/Frontend)
- Proxy an eigenen Server für Live-Demo
- Oder: Space zeigt statische Inhalte + Link zu Live-Demo

**Vorteile:**
- Best of both worlds: Discovery auf ModelScope + Kontrolle auf eigenem Server
- ModelScope-SEO + eigene Infrastruktur
- KI-Kosten kontrollierbar durch eigenen sigoREST

---

## Offene Entscheidungen

### 1. Zugangsebene
| Stufe | Beschreibung | Technische Umsetzung |
|-------|--------------|---------------------|
| Geschlossen | Nur invited Users | nginx Basic Auth + IP-Whitelist |
| Halb-offen | Jeder kann Code sehen, KI nur mit Passwort | Demo-Modus + Auth für Live-KI |
| Offen | Jeder kann alles | Rate-Limiting auf sigoREST |

### 2. Was soll demonstriert werden?
- [ ] Basis-Lisp (Arithmetik, Listen, Funktionen)
- [ ] TCO (Tail Call Optimization)
- [ ] Makros (`defmacro`, `quasiquote`)
- [ ] **parfunc / Goroutines** (besonders interessant)
- [ ] **KI-Integration** (`sigo`, Multi-Model)
- [ ] PostgreSQL-Integration
- [ ] Channels/Locks

**Hinweis:** `parfunc` mit echter KI ist das "Killer-Feature", aber auch das teuerste.

### 3. sigoREST-Beschränkungen
Mögliche Limits auf sigoREST-Seite:
- Max Requests pro IP/Session
- Nur bestimmte Modelle (z.B. nur Ollama-Modelle, keine Commercial-APIs)
- Zeitfenster (z.B. nur 09:00-18:00 CET)
- Budget-Cap pro Tag

---

## Technische Details (zur Überlegung)

### Cross-Compile für Linux
```bash
GOOS=linux GOARCH=amd64 go build -o golisp2 .
```

### nginx Konfiguration (Beispiel)
```nginx
# Basic Auth
location /golisp2 {
    auth_basic "GoLisp Demo";
    auth_basic_user_file /etc/nginx/.htpasswd;
    proxy_pass http://localhost:8080;
}

# Oder: IP-Whitelist
location /golisp2 {
    allow 203.0.113.0/24;  # ModelScope IPs?
    allow 198.51.100.0/24; # Eigene IPs
    deny all;
    proxy_pass http://localhost:8080;
}
```

### Demo-Modus in GoLisp
```lisp
; Wenn GOLISP_DEMO=1 gesetzt:
(sigo "Hallo" "claude-h")
; -> "[DEMO-MODUS] KI-Antwort würde hier erscheinen"
;    "Für Live-Demo: eigenen sigoREST konfigurieren"
```

---

## Nächste Schritte (Wenn entschieden)

1. **Architektur wählen** (A, B oder C)
2. **Zugangskonzept festlegen** (offen/halb/geschlossen)
3. **sigoREST-Limitierung implementieren**
4. **Demo-Modus in GoLisp ergänzen** (falls nötig)
5. **Deployment auf ModelScope und/oder eigenen Server**

---

## Notizen

> "Das reizvolle sind die vielen KI-Anbindungen."
> — Gerhard, 2026-02-27

Die Herausforderung: `parfunc` mit 6 KIs parallel zeigen ohne Kostenexplosion.

Möglicher Kompromiss:
- **Statische Demo:** Code + Ergebnisse als Text/Video
- **Live-Demo:** Limitierte Aufrufe, nach Registrierung/Freischaltung

---

*Dokument erstellt für Überlegungsphase. Bei biologischen Intelligenzen dauert das etwas länger. ;-)*
