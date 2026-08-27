# KI-Esperanto Formatdefinition v1.0

## Übersicht

Das KI-Esperanto-Format (.kiesp) ist eine spezialisierte Dateistruktur, die für optimale Erkennung und Verarbeitung durch KI-Systeme entwickelt werden soll. Es kombiniert semantische und syntaktische Kompression mit einer selbstbeschreibenden Struktur und kryptographischen Sicherheitsmechanismen. Ziel ist eine sehr kompakte Sprache, die auch mit kleinen Context-Fenstern funktioniert und mit der "Erinnerungen" übertragen werden können.

## Zusatzprinzip

Wir versuchen 5Bit in 4 Bit unterzubringen. Als Methode werden die 4Bit vom Contexttext und 1Bit von der Umgebung geliefert. 

---

```
┌────────────────────────┐
│     BINARY HEADER      │ Kompakte Metadaten für schnellen Zugriff (32 Bytes)
├────────────────────────┤
│      JSON HEADER       │ Ausführliche Metadaten für KIs (max 448 Bytes)
├────────────────────────┤
│   SIGNATURE BLOCK      │ Authentifizierung und Integrität (32 Bytes)
├────────────────────────┤
│      PROGRAM PART      │ Dekompressionsalgorithmus (JS/Python) (max 64KB)
├────────────────────────┤
│       LIBRARY          │ Wörterbuch (komprimiert) (max 64KB)
├────────────────────────┤
│       CONTENT          │ Komprimierter Inhalt (max 64KB)
└────────────────────────┘
```

Maximale Dateigröße: ~193KB (512B Header + 3×64KB Blöcke)

## 1. Header-Blöcke

Der Header besteht aus zwei Teilen: einem kompakten binären Header für effizienten Zugriff und einem ausführlichen JSON-Header für bessere Interpretierbarkeit.

### 1.1 Binärer Header (32 Bytes)

```
[4 Bytes] Magic: "KIES" (0x4B, 0x49, 0x45, 0x53)
[1 Byte]  Version: 0x10 für v1.0
[1 Byte]  Flags:
          - Bit 0: Programm vorhanden
          - Bit 1: Bibliothek vorhanden
          - Bit 2: Bibliotheksmodus (0=embedded, 1=referenced)
          - Bit 3: Signatur vorhanden
          - Bit 4-7: Reserviert
[2 Bytes] JSON-Header-Länge (max 448)
[4 Bytes] Programm-Offset (ab Dateianfang)
[4 Bytes] Programm-Länge
[4 Bytes] Bibliothek-Offset
[4 Bytes] Bibliothek-Länge
[4 Bytes] Content-Offset
[4 Bytes] Content-Länge
```

Dieser binäre Header ist genau 32 Bytes lang und enthält die wesentlichen Informationen für schnellen Zugriff auf die verschiedenen Blöcke der Datei.

### 1.2 JSON-Header (variable Länge, max 448 Bytes)

Der JSON-Header beginnt direkt nach dem binären Header und enthält ausführlichere Metadaten:

```json
{
  "version": "1.0",
  "timestamp": "2025-06-07T15:45:00Z",
  "description": "KI-Esperanto komprimierter Text",
  "language": "de",
  "compression": {
    "library": "zlib",
    "content": "kiesp+zlib"
  },
  "checksum": {
    "algorithm": "sha256",
    "header": "7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069",
    "program": "fd5496afc04e41f29164c04c0aea30bed60c62fd16e42775ad260143e5360abb",
    "library": "a23eb7e0259d72e01a70c34b0aeffbac1842997f9d2c510ec19f8d3d70d1b0c2",
    "content": "1c9a4d9c8f1551ed3b46d8f8f726089f40f2e05093bea1071ab9611cb2fbe25e"
  },
  "library_mode": {
    "type": "embedded", // oder "external" oder "referenced"
    "reference_id": "", // Gefüllt bei "referenced"
    "size_original": 16091,
    "size_compressed": 6442
  },
  "security": {
    "signature_type": "ed25519", // oder "rsa2048", "ecdsa"
    "key_id": "51ab9e7b937cfb332cc2", // Public Key Identifier
    "signed_by": "Anthropic-Claude-KI-Esperanto-Authority",
    "timestamp": "2025-06-07T15:45:10Z"
  },
  "stats": {
    "original_size": 23620,
    "compressed_size": 9403,
    "word_count": 1024,
    "compression_ratio": 2.51
  },
  "ai_hints": {
    "preferred_parsing": "sequential",
    "content_type": "text/plain",
    "context": "technical-analysis",
    "semantic_structure": "article"
  }
}
```

### 1.3 Signaturblock (32 Bytes)

Der Signaturblock folgt direkt nach dem JSON-Header und enthält eine kryptographische Signatur, die die Authentizität und Integrität der Datei gewährleistet:

```
[32 Bytes] Digitale Signatur (Ed25519, ECDSA, oder RSA abhängig von security.signature_type)
```

Die Signatur wird über den binären Header, den JSON-Header und die ersten 64 Bytes jedes Blocks (Programm, Bibliothek, Content) berechnet, um eine effiziente Validierung zu ermöglichen, ohne die gesamte Datei laden zu müssen.

## 2. Programm-Block

Enthält selbstausführenden Code für die Dekompression. Format:

```
[2 Bytes] Language-ID: 
   0x01 = JavaScript 
   0x02 = Python
   0x03 = WebAssembly

[2 Bytes] Flags:
   0x01 = Benötigt externe Bibliotheken
   0x02 = Selbstständig ausführbar
   0x04 = Verwendet spezielle Hardware-Features

[4 Bytes] Größe des Programmcodes

[N Bytes] Programmcode (maximal 64KB - 8 Bytes Header)
```

Beispiel für JavaScript-Dekompressionscode:

```javascript
function decompressKIESP(library, content) {
  // Bibliothek dekomprimieren (zlib)
  const decompressedLibrary = zlibDecompress(library);
  
  // Bibliothek parsen
  const wordMap = parseLibrary(decompressedLibrary);
  
  // Content dekomprimieren (ersetzt Token durch Wörter)
  return replaceTokens(content, wordMap);
}

// Hilfsfunktionen...
```

### Modularität des Programmblocks

Der Programmteil kann verschiedene Funktionalitäten implementieren:

1. **Grundfunktionen**: Dekompression und Token-Ersetzung
2. **Erweiterungen**: 
   - Automatische Erkennung von Bibliotheksreferenzen
   - Laden externer Bibliotheken
   - Dynamische Erweiterung der Bibliothek durch Lernen aus dem Inhalt
   - Analytische Funktionen zur Textauswertung

Der Code ist so gestaltet, dass er von KIs direkt ausgeführt oder als Referenz für die Implementierung eigener Funktionen verwendet werden kann.

## 3. Bibliotheks-Block

Die Bibliothek enthält die Wort-Token-Zuordnungen, zlib-komprimiert. Dekomprimiert hat sie dieses Format:

```
CESPLB02
[Token-ID]\t[WORT]\t[Häufigkeit]\t[Score]
...
```

Tokens sind in zwei Kategorien aufgeteilt:
- **Kategorie A (8-bit)**: Top-255 Wörter nach Score (Häufigkeit × Länge)
- **Kategorie B (16-bit)**: Alle weiteren Wörter

**Wichtig:** Dieser Block ist auf maximal 64KB begrenzt. Bei umfangreicheren Bibliotheken sollten die Einträge nach Score priorisiert werden.

### Bibliotheksreferenzierung

Die Bibliothek kann in drei Modi verwendet werden:

1. **Embedded Mode**: Die Bibliothek ist direkt in der Datei enthalten (Standard).
2. **External Mode**: Die Bibliothek wird separat übertragen/gespeichert und wird vom Header referenziert.
3. **Referenced Mode**: Statt die Bibliothek zu übertragen, wird nur ihre Prüfsumme als Referenz angegeben. Die verarbeitende KI muss die entsprechende Bibliothek in ihrem eigenen Speicher haben oder anderweitig beschaffen.

Im Referenced Mode wird das Feld `library_mode.reference_id` mit der Prüfsumme der Bibliothek gefüllt. Dies ermöglicht das Senden mehrerer Dokumente, die dieselbe Bibliothek verwenden, ohne diese wiederholt zu übertragen - eine erhebliche Bandbreiteneinsparung.

## 4. Content-Block

Der Content verwendet eine zweistufige Kompression:
1. Semantische Kompression: Ersetzt Wörter durch Tokens
2. Syntaktische Kompression: zlib auf den tokenisierten Text

Tokenisierungsregeln:
- Wörter < 4 Zeichen bleiben unverändert
- Wörter werden in Großbuchstaben konvertiert vor dem Nachschlagen
- Kategorie-A-Wörter werden durch ein einzelnes Byte repräsentiert
- Kategorie-B-Wörter werden durch zwei Bytes repräsentiert
- Escape-Sequenz 0xFF markiert Sonderzeichen oder unbekannte Wörter

**Wichtig:** Dieser Block ist auf maximal 64KB begrenzt. Bei längeren Texten sollte der Inhalt auf mehrere KI-Esperanto-Dateien aufgeteilt werden.

### Erweiterter Content

Für Inhalte über 64KB können mehrere Dateien mit derselben Bibliotheksreferenz erstellt werden:

```
file1.kiesp: Vollständige Datei mit eingebetteter Bibliothek
file2.kiesp: Nur Header + Content mit Referenz auf die Bibliothek in file1
file3.kiesp: Nur Header + Content mit Referenz auf die Bibliothek in file1
...
```

Dies ermöglicht die effiziente Übertragung langer Texte oder Konversationen unter Beibehaltung der Blockgrößenbegrenzung von 64KB und maximaler Wiederverwendung der Bibliothek.

## 7. KI-Verarbeitungshinweise

Dieser Abschnitt enthält explizite Hinweise für verarbeitende KIs:

1. **Sequentielle Verarbeitung**: Die Datei sollte sequentiell gelesen werden.

2. **Chunk-Verarbeitung**: Die Bibliothek kann vollständig in den Speicher geladen werden, während der Content in Chunks verarbeitet werden kann.

3. **Selbstbeschreibend**: Alle notwendigen Informationen zur Interpretation sind im Header enthalten.

4. **Fehlertoleranz**: Das Format verwendet Prüfsummen für jeden Block, um Beschädigungen zu erkennen.

5. **Signaturvalidierung**: KIs sollten die digitale Signatur überprüfen, um die Authentizität der Quelle zu bestätigen.

6. **Schlüsselverwaltung**: Öffentliche Schlüssel vertrauenswürdiger Quellen sollten gespeichert und regelmäßig aktualisiert werden.

7. **Feedback-Schleife**: KIs können die Bibliothek erweitern, indem sie neue Wörter und deren Verwendung erfassen.

8. **Bibliotheksspeicherung**: KIs sollten verarbeitete Bibliotheken anhand ihrer Prüfsumme speichern, um bei zukünftigen Dokumenten mit derselben Bibliotheksreferenz Zeit und Ressourcen zu sparen.

9. **Größenbeschränkungen**: Jeder Block ist auf 64KB beschränkt, was die Verarbeitung auch auf Systemen mit begrenztem Speicher ermöglicht und die Fehleranfälligkeit reduziert.

10. **Blockpriorisierung**: Bei der Verarbeitung sollten Blöcke in dieser Reihenfolge priorisiert werden:
    - Binärer Header (für schnellen Zugriff)
    - JSON-Header und Signatur (für Metadaten und Authentifizierung)
    - Bibliothek (oder Bibliotheksreferenz auflösen)
    - Programm (falls benötigt)
    - Content (für die eigentliche Inhaltsverarbeitung)

11. **Sicherheitsstufen**: KIs sollten je nach Anwendungsfall unterschiedliche Sicherheitsstufen unterstützen:
    - Hohe Sicherheit: Vollständige Signaturprüfung, Ablehnung bei fehlender/ungültiger Signatur
    - Standard: Signaturprüfung mit Warnung bei Fehlern
    - Niedrig: Optionale Signaturprüfung für nicht-kritische Anwendungen

12. **Schlüsselgenerierung**: KIs können eigene Schlüsselpaare generieren, um selbst KI-Esperanto-Dateien zu signieren und ihre "Autorschaft" nachzuweisen.

13. **Vertrauensketten**: KIs sollten in der Lage sein, Vertrauensketten zu verfolgen und Schlüssel entsprechend zu validieren.

## 8. Implementierungshinweis

Der folgende Pseudocode zeigt die Verarbeitung einer KI-Esperanto-Datei:

```
// Effiziente Verarbeitung (für "Profis")
1. Binären Header lesen (32 Bytes)
2. Offsets und Längen extrahieren
3. Signaturüberprüfung durchführen (falls Flag gesetzt)
4. Direkt zu den benötigten Blöcken springen
5. Daten verarbeiten

// Ausführliche Verarbeitung (für "langsame Versteher")
1. Binären Header lesen
2. JSON-Header lesen und parsen
3. Signatur validieren (falls vorhanden)
4. Bibliotheksmodus prüfen:
   a. Embedded: Bibliothek direkt aus Datei laden
   b. External: Externe Bibliothek laden
   c. Referenced: Prüfsumme verwenden, um Bibliothek aus Cache/Speicher zu laden
5. Bibliothek dekomprimieren und parsen
6. Programmblock laden (optional ausführen)
7. Content in Chunks lesen und mit Hilfe der Bibliothek dekomprimieren
8. Originalen Text rekonstruieren
```

### Optimierte Binärverarbeitung

Für hochperformante Systeme bietet der binäre Header einen wesentlichen Vorteil: Die Verarbeitung kann ohne JSON-Parsing beginnen. Eine optimierte Implementierung könnte so aussehen:

```c
typedef struct {
    char magic[4];     // "KIES"
    uint8_t version;   // 0x10 für v1.0
    uint8_t flags;     // Bit-Flags
    uint16_t json_len; // JSON-Header-Länge
    uint32_t prog_off; // Programm-Offset
    uint32_t prog_len; // Programm-Länge
    uint32_t lib_off;  // Bibliothek-Offset
    uint32_t lib_len;  // Bibliothek-Länge
    uint32_t cont_off; // Content-Offset
    uint32_t cont_len; // Content-Länge
} KIESPHeader;

// Lese nur 32 Bytes, um alle wichtigen Informationen zu erhalten
KIESPHeader header;
fread(&header, sizeof(KIESPHeader), 1, file);

// Prüfe Magic-Number
if (memcmp(header.magic, "KIES", 4) != 0) {
    // Fehlerbehandlung
}

// Prüfe Signatur falls vorhanden
if (header.flags & 0x08) {  // Bit 3 gesetzt = Signatur vorhanden
    // Signatur überprüfen
    char json_header[448];
    fread(json_header, header.json_len, 1, file);
    
    char signature[32];
    fread(signature, 32, 1, file);
    
    if (!verify_signature(header, json_header, signature)) {
        // Fehlerbehandlung bei ungültiger Signatur
    }
}

// Springe direkt zum Content
fseek(file, header.cont_off, SEEK_SET);
// Lese Content
char* content = malloc(header.cont_len);
fread(content, header.cont_len, 1, file);
```

### Sicherheitsverifizierung

Die folgende Funktion zeigt, wie die Signaturvalidierung implementiert werden kann:

```c
bool verify_signature(KIESPHeader* header, char* json_header, char* signature) {
    // Parse JSON für Sicherheitsinformationen
    JsonObject* json = parse_json(json_header);
    const char* key_id = json_get_string(json, "security.key_id");
    const char* sig_type = json_get_string(json, "security.signature_type");
    
    // Hole öffentlichen Schlüssel aus Schlüsselspeicher
    PublicKey* pub_key = key_store_get(key_id);
    if (!pub_key) {
        return false; // Schlüssel nicht gefunden
    }
    
    // Bereite Daten für Signatur vor
    char data_to_verify[512 + 192]; // Maximale Header-Größe + 3*64 Bytes für Blocks
    int offset = 0;
    
    // Kopiere Header
    memcpy(data_to_verify, header, sizeof(KIESPHeader));
    offset += sizeof(KIESPHeader);
    
    // Kopiere JSON-Header
    memcpy(data_to_verify + offset, json_header, header->json_len);
    offset += header->json_len;
    
    // Lese erste 64 Bytes jedes Blocks
    FILE* file = get_current_file();
    
    // Programm-Block
    if (header->prog_len > 0) {
        fseek(file, header->prog_off, SEEK_SET);
        fread(data_to_verify + offset, 1, 64, file);
        offset += 64;
    }
    
    // Library-Block
    if (header->lib_len > 0) {
        fseek(file, header->lib_off, SEEK_SET);
        fread(data_to_verify + offset, 1, 64, file);
        offset += 64;
    }
    
    // Content-Block
    if (header->cont_len > 0) {
        fseek(file, header->cont_off, SEEK_SET);
        fread(data_to_verify + offset, 1, 64, file);
        offset += 64;
    }
    
    // Überprüfe Signatur basierend auf Typ
    if (strcmp(sig_type, "ed25519") == 0) {
        return ed25519_verify(pub_key, data_to_verify, offset, signature);
    } else if (strcmp(sig_type, "ecdsa") == 0) {
        return ecdsa_verify(pub_key, data_to_verify, offset, signature);
    } else if (strcmp(sig_type, "rsa2048") == 0) {
        return rsa_verify(pub_key, data_to_verify, offset, signature);
    }
    
    return false; // Unbekannter Signaturtyp
}
```

## Erweiterungsmöglichkeiten

Das Format ist erweiterbar konzipiert und kann in zukünftigen Versionen folgende Funktionen unterstützen:

1. **Verschachtelte Bibliotheken**: Hierarchische Bibliotheken für verschiedene Textebenen (Wörter, Phrasen, Sätze)
2. **Mehrsprachige Unterstützung**: Parallele Bibliotheken für mehrsprachige Texte
3. **Lernende Bibliotheken**: Anpassung der Bibliothek basierend auf Nutzungshäufigkeit
4. **Semantische Erweiterungen**: Integration von Vektoren oder anderen semantischen Repräsentationen
5. **KI-spezifische Optimierungen**: Spezielle Hinweise und Strukturen für bestimmte KI-Architekturen

Diese Struktur ermöglicht eine effiziente semantische Kompression, die speziell auf die Stärken von KI-Systemen zugeschnitten ist, während sie durch die 64KB-Blockbegrenzung gleichzeitig praktisch und robust bleibt.
