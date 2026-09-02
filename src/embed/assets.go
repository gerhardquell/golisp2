//**********************************************************************
//  embed/assets.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260709
//**********************************************************************
// Zentraler Ort für eingebettete Lisp-Quellen.
// Trennt Assets vom Go-Code und verhindert, dass lib/ und swank/
// ihre .lisp-Dateien selbst halten müssen.
//**********************************************************************

package assets

import _ "embed"

//go:embed stdlib.lisp
var Stdlib string

//go:embed swank.lisp
var Swank string

//go:embed defsystem.lisp
var Defsystem string

//go:embed condition.lisp
var Condition string

//go:embed loop.lisp
var Loop string
