//**********************************************************************
//  lib/postgres.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260224
//**********************************************************************
// PostgreSQL-Integration für GoLisp
//**********************************************************************

package lib

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// pgConn verwaltet eine PostgreSQL-Verbindung
type pgConn struct {
	db *sql.DB
}

// RegisterPostgres registriert alle Postgres-Funktionen in der Umgebung
func RegisterPostgres(env *Env) {
	_ = env.Set("pg-connect", makeFn(fnPgConnect))
	_ = env.Set("pg-query", makeCtxFn(fnPgQuery))
	_ = env.Set("pg-exec", makeCtxFn(fnPgExec))
	_ = env.Set("pg-close", makeFn(fnPgClose))
}

// pg-connect: (pg-connect "host=... port=... user=... password=... dbname=...")
func fnPgConnect(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("pg-connect: connection string required")
	}

	connStr := args[0].Val
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("pg-connect: %w", err)
	}

	// Teste die Verbindung
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pg-connect: ping failed: %w", err)
	}

	return MakeForeign("pg-connection", &pgConn{db: db}), nil
}

// pg-query: (pg-query conn "SELECT * FROM users WHERE id = $1" 42)
// Gibt eine Liste von Zeilen zurück, jede Zeile ist eine Assoziationsliste
func fnPgQuery(args []*Cell, ectx evalCtx) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("pg-query: connection and query required")
	}

	// Extrahiere Connection
	conn, ok := asPgConn(args[0])
	if !ok {
		return nil, fmt.Errorf("pg-query: not a postgres connection")
	}

	// Extrahiere Query
	query := args[1].Val

	// Sammle Query-Parameter (ab Index 2)
	var params []interface{}
	for i := 2; i < len(args); i++ {
		params = append(params, cellToInterface(args[i]))
	}

	// Führe Query aus
	ctx := pgContext(ectx)
	rows, err := conn.db.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, pgErr(ctx, "pg-query", err)
	}
	defer rows.Close()

	// Lese Spaltennamen
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("pg-query: %w", err)
	}

	// Ergebniszeilen sammeln
	var resultRows []*Cell

	for rows.Next() {
		// Scan-Werte in slice
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("pg-query: scan error: %w", err)
		}

		// Erstelle Assoziationsliste für diese Zeile
		var rowCells []*Cell
		for i, col := range columns {
			val := interfaceToCell(values[i])
			// Paar: ("column" . value)
			pair := Cons(MakeAtom(col), val)
			rowCells = append(rowCells, pair)
		}

		// Zeile als Liste hinzufügen
		resultRows = append(resultRows, SliceToCell(rowCells))
	}

	if err := rows.Err(); err != nil {
		return nil, pgErr(ctx, "pg-query", err)
	}

	return SliceToCell(resultRows), nil
}

// pg-exec: (pg-exec conn "INSERT INTO users (name) VALUES ($1)" "Alice")
// Gibt die Anzahl der betroffenen Zeilen zurück
func fnPgExec(args []*Cell, ectx evalCtx) (*Cell, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("pg-exec: connection and query required")
	}

	// Extrahiere Connection
	conn, ok := asPgConn(args[0])
	if !ok {
		return nil, fmt.Errorf("pg-exec: not a postgres connection")
	}

	// Extrahiere Query
	query := args[1].Val

	// Sammle Query-Parameter
	var params []interface{}
	for i := 2; i < len(args); i++ {
		params = append(params, cellToInterface(args[i]))
	}

	// Führe Query aus
	ctx := pgContext(ectx)
	result, err := conn.db.ExecContext(ctx, query, params...)
	if err != nil {
		return nil, pgErr(ctx, "pg-exec", err)
	}

	// Anzahl betroffener Zeilen
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("pg-exec: %w", err)
	}

	return MakeInt(affected), nil
}

// pgContext: Abbruch-Kontext des Aufrufers (Sandbox-Frist, parfunc
// :timeout), ohne Kontext Background.
func pgContext(ectx evalCtx) context.Context {
	if ctx := ectx.context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// pgErr: bei Abbruch derselbe Fehler wie check(), sonst die Treibermeldung.
func pgErr(ctx context.Context, name string, err error) error {
	if ctx.Err() != nil {
		return cancelledErr()
	}
	return fmt.Errorf("%s: %w", name, err)
}

// pg-close: (pg-close conn)
func fnPgClose(args []*Cell) (*Cell, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("pg-close: connection required")
	}

	if obj, ok := ForeignObj(args[0], "pg-connection"); args[0].Type == NIL || ok && obj == nil {
		return MakeNil(), nil // bereits geschlossen
	}
	conn, ok := asPgConn(args[0])
	if !ok {
		return nil, fmt.Errorf("pg-close: not a postgres connection")
	}

	if err := conn.db.Close(); err != nil {
		return nil, fmt.Errorf("pg-close: %w", err)
	}

	// Markiere als geschlossen
	args[0].Env = nil
	return MakeAtom("t"), nil
}

// cellToInterface konvertiert eine Cell zu einem Go-Interface{}
func cellToInterface(c *Cell) interface{} {
	if c == nil {
		return nil
	}
	switch c.Type {
	case NUMBER:
		if c.NumKind == NumInt {
			return c.Int
		}
		return c.Num
	case STRING:
		return c.Val
	case ATOM:
		return c.Val
	case NIL:
		return nil
	default:
		return c.String()
	}
}

// interfaceToCell konvertiert einen Go-Wert zu einer Cell
func interfaceToCell(v interface{}) *Cell {
	if v == nil {
		return MakeNil()
	}

	switch val := v.(type) {
	case int64:
		return MakeInt(val)
	case int32:
		return MakeInt(int64(val))
	case int:
		return MakeInt(int64(val))
	case float64:
		return MakeNum(val)
	case float32:
		return MakeNum(float64(val))
	case string:
		return MakeStr(val)
	case bool:
		if val {
			return MakeAtom("t")
		}
		return MakeNil()
	case []byte:
		return MakeStr(string(val))
	case time.Time:
		return MakeStr(val.Format(time.RFC3339))
	default:
		return MakeStr(fmt.Sprintf("%v", val))
	}
}

func asPgConn(c *Cell) (*pgConn, bool) {
	obj, ok := ForeignObj(c, "pg-connection")
	if !ok {
		return nil, false
	}
	conn, ok := obj.(*pgConn)
	return conn, ok
}
