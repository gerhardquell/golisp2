# Design: Shared-Memory-Integration in GoLisp2

**Datum:** 2026-07-11  
**Status:** Genehmigt  
**Autor:** Gerhard Quell / Claude  

## Ziel

Die bestehende Shared-Memory-Bibliothek `lib/shm` (SysV SHM + Pool-Manager) soll als Lisp-Primitive in GoLisp2 verfügbar gemacht werden. Es wird ausschließlich die **High-Level Pool-API** exponiert.

## Architektur

- Neue Datei `lib/shm_lisp.go` im Hauptpaket `lib`
- Registrierung via `RegisterShm(env *Env)` in `BaseEnv()`
- SHM-Handles werden als opake Cells verpackt, analog zu Channels (`chan-make`) und Mutexes (`lock-make`)
- Der globale Pool (`shm.GetPool()`) wird bei erstem `shm-alloc` lazy initialisiert

## Lisp-API

| Primitive | Signatur | Rückgabe |
|-----------|----------|----------|
| `shm-alloc` | `(shm-alloc)` | Handle `#<shm-block N>` |
| `shm-free` | `(shm-free handle)` | `t` |
| `shm-write` | `(shm-write handle string)` | Geschriebener String |
| `shm-read` | `(shm-read handle [len])` | String |
| `shm-status` | `(shm-status)` | `t` (druckt Tabelle) |
| `shm-cleanup` | `(shm-cleanup)` | `t` |

### Semantik

- `(shm-read handle)` liest bis zum ersten NUL-Byte (`0x00`).
- `(shm-read handle 256)` liest genau 256 Bytes, auf `shm.POOL_SIZE` begrenzt.
- `(shm-status)` gibt die Belegung aller Pool-Blöcke aus.
- `(shm-cleanup)` trennt alle Blöcke (`ShmDt`) und entfernt die Segmente (`ShmRm`).

## Go-Implementierung

### Handle-Repräsentation

```go
type shmHandle struct {
  block *shm.ShmBlock
}

func makeShmCell(block *shm.ShmBlock) *Cell {
  return &Cell{
    Type: FUNC,
    Val:  "shm-block",
    Env:  &shmHandle{block: block},
  }
}

func getShmBlock(c *Cell) (*shm.ShmBlock, error) {
  if c == nil || c.Val != "shm-block" {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  h, ok := c.Env.(*shmHandle)
  if !ok { return nil, fmt.Errorf("kein SHM-Block") }
  return h.block, nil
}
```

### Registrierung

```go
func RegisterShm(env *Env) {
  env.Set("shm-alloc",   makeFn(fnShmAlloc))
  env.Set("shm-free",    makeFn(fnShmFree))
  env.Set("shm-write",   makeFn(fnShmWrite))
  env.Set("shm-read",    makeFn(fnShmRead))
  env.Set("shm-status",  makeFn(fnShmStatus))
  env.Set("shm-cleanup", makeFn(fnShmCleanup))
}
```

## Fehlerbehandlung

Alle Fehler lösen eine Go-Ausnahme aus (`return nil, fmt.Errorf("shm-xxx: ...")`), konsistent mit `file-read`, `chan-send` etc.

Mögliche Fehler:
- Ungültiger Handle → `shm-xxx: kein SHM-Block`
- Pool erschöpft → `shm-alloc: ERR102: No free SHM blocks`
- Ungültige Block-ID (intern) → wird vom Pool geprüft

## Testansatz

- Go-Unit-Test in `lib/shm_lisp_test.go`
- Testfälle:
  1. `shm-alloc` + `shm-write` + `shm-read` Roundtrip
  2. `shm-read` ohne Länge stoppt am NUL-Byte
  3. `shm-read` mit expliziter Länge
  4. `shm-free` danach erneutes `shm-alloc` möglich
  5. Ungültiger Handle führt zu Fehler
  6. `shm-cleanup` hinterlässt keine Lecks

## Dateien

| Datei | Änderung |
|-------|----------|
| `lib/shm_lisp.go` | neu |
| `lib/primitives.go` | `RegisterShm(env)` in `BaseEnv()` einfügen |
| `lib/shm_lisp_test.go` | neu |
