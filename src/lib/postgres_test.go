//**********************************************************************
//  lib/postgres_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-opus-5.5
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261118
//**********************************************************************
// Postgres gegen eine lokale Test-DB: Verbindungs-String aus
// GOLISP2_PG_TEST (z. B. "dbname=golisp2lernen_test sslmode=disable"),
// ohne Variable übersprungen. Nur SELECTs, nichts Dauerhaftes.
//**********************************************************************

package lib

import (
  "context"
  "fmt"
  "os"
  "testing"
  "time"
)

func pgTestEnv(t *testing.T) *Env {
  t.Helper()
  dsn := os.Getenv("GOLISP2_PG_TEST")
  if dsn == "" {
    t.Skip("GOLISP2_PG_TEST nicht gesetzt")
  }
  env := sbEnv(t, "(make-sandbox :zeitlimit 1 :erlaube '(pg-connect pg-query pg-close))")
  if _, err := sbRun(env, fmt.Sprintf("(define dsn %q) (define db (pg-connect dsn))", dsn)); err != nil {
    t.Fatalf("pg-connect: %v", err)
  }
  t.Cleanup(func() { _, _ = sbRun(env, "(pg-close db)") })
  return env
}

// Ganzzahlen kommen exakt zurück und gehen als int64 an den Treiber.
func TestPgZahlen(t *testing.T) {
  env := pgTestEnv(t)
  sbWant(t, env, `(pg-query db "select 7::int2 as a, 70000::int4 as b, 9007199254740993::int8 as c, 1.5::float8 as f")`,
    "(((a . 7) (b . 70000) (c . 9007199254740993) (f . 1.5)))")
  sbWant(t, env, `(let ((z (car (pg-query db "select 9007199254740993::int8 as c, 1.5::float8 as f"))))
                    (list (integerp (cdr (assoc 'c z))) (floatp (cdr (assoc 'f z)))))`, "(t t)")
  sbWant(t, env, `(pg-query db "select $1::int8 + 0 as c, $2::float8 as f" 9007199254740993 2.5)`,
    "(((c . 9007199254740993) (f . 2.5)))")
}

// Abbruch-Kontext mit Frist: pg-query kehrt nach der Frist zurück,
// mit dem Fehler von cancelledErr statt der Treibermeldung.
func TestPgQueryAbbruch(t *testing.T) {
  env := pgTestEnv(t)
  for _, name := range []string{"pg-query", "pg-exec"} {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    form, err := Read(`(` + name + ` db "select pg_sleep(5)")`)
    if err != nil {
      t.Fatal(err)
    }
    start := time.Now()
    _, err = evalWithCtx(form, env, evalCtx{run: &runState{ctx: ctx}})
    d := time.Since(start)
    cancel()
    t.Logf("%s mit Frist 1 s: Abbruch nach %v", name, d)
    if err == nil || err.Error() != cancelledErr().Error() {
      t.Errorf("%s: Fehler %v, want %v", name, err, cancelledErr())
    }
    if d > 3*time.Second {
      t.Errorf("%s: Abbruch erst nach %v", name, d)
    }
  }
}

// In der Sandbox: Zeitlimit greift, und das Statement läuft auf dem
// Server nicht weiter.
func TestPgSandboxZeitlimit(t *testing.T) {
  env := pgTestEnv(t)
  start := time.Now()
  _, err := sbRun(env, `(sandbox-eval sb (list 'let (list (list 'c (list 'pg-connect dsn)))
                          '(pg-query c "select pg_sleep(5) /* golisp2-sbtest */")))`)
  d := time.Since(start)
  t.Logf("sandbox-eval :zeitlimit 1: Abbruch nach %v", d)
  if err == nil || err.Error() != "sandbox: Zeitlimit 1 s überschritten" {
    t.Errorf("Fehler %v", err)
  }
  if d > 3*time.Second {
    t.Errorf("Abbruch erst nach %v", d)
  }
  // Treiber-Cancel braucht einen Moment bis zum Server.
  for i := 0; ; i++ {
    res, err := sbRun(env, `(pg-query db "select count(*) as n from pg_stat_activity
                              where state = 'active' and query like '%golisp2-sbtest */'")`)
    if err != nil {
      t.Fatal(err)
    }
    if res.String() == "(((n . 0)))" {
      break
    }
    if i == 20 {
      t.Fatalf("Statement läuft auf dem Server weiter: %s", res)
    }
    time.Sleep(100 * time.Millisecond)
  }
}

// Ohne DB: eine normale Liste ist keine Verbindung. Mit DB: das Handle
// ist FOREIGN.
func TestPgConnectionIstForeign(t *testing.T) {
  for name, err := range map[string]error{
    "pg-query": second(fnPgQuery([]*Cell{keinHandle, MakeStr("select 1")}, evalCtx{})),
    "pg-exec":  second(fnPgExec([]*Cell{keinHandle, MakeStr("select 1")}, evalCtx{})),
    "pg-close": second(fnPgClose([]*Cell{keinHandle})),
  } {
    if err == nil || err.Error() != name+": not a postgres connection" {
      t.Errorf("%s mit Liste: err = %v", name, err)
    }
  }
  dsn := os.Getenv("GOLISP2_PG_TEST")
  if dsn == "" {
    t.Skip("GOLISP2_PG_TEST nicht gesetzt")
  }
  conn, err := fnPgConnect([]*Cell{MakeStr(dsn)})
  if err != nil {
    t.Fatalf("pg-connect: %v", err)
  }
  t.Cleanup(func() { _, _ = fnPgClose([]*Cell{conn}) })
  pruefeHandle(t, conn, "pg-connection")
}

func second(_ *Cell, err error) error { return err }
