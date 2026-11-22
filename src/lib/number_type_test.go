//**********************************************************************
//  lib/number_type_test.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude-fable-5-2
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20261116
//**********************************************************************
// Ganzzahl-Typ, Spec 2026-11-16 §3.4: Typen und Umwandlung nach echter
// Zahlart. Dazu: Druckform großer Floats und n-äres /.
//**********************************************************************

package lib

import "testing"

func TestIntTypen(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    // integerp, floatp, float
    {`(integerp 3)`, "t"},
    {`(integerp 3.0)`, "()"},
    {`(integerp "3")`, "()"},
    {`(floatp 3.0)`, "t"},
    {`(floatp 3)`, "()"},
    {`(floatp 'x)`, "()"},
    {`(float 3)`, "3.0"},
    {`(float 3.5)`, "3.5"},
    {`(floatp (float 3))`, "t"},
    {`(float "3")`, "ERR:float: Zahl erwartet"},
    {`(integerp (floor (float 7)))`, "t"},
    // type-of, typep, subtypep
    {`(type-of 3)`, "integer"},
    {`(type-of 3.0)`, "float"},
    {`(type-of (/ 6 2))`, "integer"},
    {`(type-of (/ 7 2))`, "float"},
    {`(type-of 9007199254740993)`, "integer"},
    {`(typep 3 'integer)`, "t"},
    {`(typep 3.0 'integer)`, "()"},
    {`(typep 3.0 'float)`, "t"},
    {`(typep 3 'float)`, "()"},
    {`(typep 3 'rational)`, "t"},
    {`(typep 3.0 'rational)`, "()"},
    {`(typep 3.0 'real)`, "t"},
    {`(subtypep 'integer 'real)`, "t"},
    {`(subtypep 'integer 'float)`, "()"},
    // Bereichstypen
    {`(typep 3 '(integer 0 5))`, "t"},
    {`(typep 6 '(integer 0 5))`, "()"},
    {`(typep 3.0 '(integer 0 5))`, "()"},
    {`(typep 5 '(integer 0 (5)))`, "()"},
    {`(typep 3.0 '(float 0 5))`, "t"},
    {`(typep 3 '(float 0 5))`, "()"},
    {`(typep 3 '(real 0 5))`, "t"},
    {`(typep 9007199254740993 '(integer 9007199254740993 *))`, "t"},
    {`(typep 9007199254740992 '(integer 9007199254740993 *))`, "()"},
    {`(subtypep '(integer 0 5) '(integer 0 (6)))`, "t"},
    // typecase, check-type
    {`(typecase 3 (float 'f) (integer 'i))`, "i"},
    {`(typecase 3.0 (integer 'i) (float 'f))`, "f"},
    {`(typecase 7 ((integer 0 5) 'klein) (integer 'gross))`, "gross"},
    {`(let ((x 3)) (check-type x integer))`, "()"},
    {`(let ((x 3.0)) (check-type x integer))`, "ERR:check-type: x ist 3.0"},
    {`(let ((x 9)) (check-type x (integer 0 5)))`, "ERR:check-type: x ist 9"},
    // parse-int, parse-float, string->number
    {`(parse-int "42")`, "42"},
    {`(parse-int "9007199254740993")`, "9007199254740993"},
    {`(parse-int "3.0")`, "ERR:parse-int"},
    {`(parse-int "9223372036854775808")`, "ERR:parse-int"},
    {`(parse-float "3")`, "3.0"},
    {`(parse-float "3.5")`, "3.5"},
    {`(string->number "42")`, "42"},
    {`(string->number "-7")`, "-7"},
    {`(string->number "3.0")`, "3.0"},
    {`(string->number "1e3")`, "1000.0"},
    {`(string->number "1.")`, "1.0"},
    {`(string->number "9223372036854775808")`, "ERR:außerhalb des int64-Bereichs"},
    {`(string->number "abc")`, "ERR:string->number"},
  })
}

// Ganzzahliger Float: Ziffern + ".0" nur für |x| < 1e16, darüber Exponent
// (Festkomma täuschte Stellen vor). Die Druckform liest wieder als Float.
func TestFloatDruckformGross(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`1e15`, "1000000000000000.0"},
    {`9999999999999998.0`, "9999999999999998.0"},
    {`1e16`, "1e+16"},
    {`1e17`, "1e+17"},
    {`-1e17`, "-1e+17"},
    {`1.5e17`, "1.5e+17"},
    {`(float 100000000000000000)`, "1e+17"},
    {`1e+17`, "1e+17"},
    {`(floatp 1e+17)`, "t"},
    {`(floatp (string->number (number->string 1e17)))`, "t"},
    {`(= 1e17 (string->number (number->string 1e17)))`, "t"},
    {`(floatp (string->number (number->string 3.0)))`, "t"},
    {`100000000000000000`, "100000000000000000"},
  })
}

func TestDivNaer(t *testing.T) {
  runNumCases(t, []struct{ src, want string }{
    {`(/ 12 2 3)`, "2"},
    {`(/ 12 2 3 2)`, "1"},
    {`(/ 12 2 4)`, "1.5"},
    {`(/ 7 2 7)`, "0.5"},
    {`(/ 12 2.0 3)`, "2.0"},
    {`(/ 12 2 3.0)`, "2.0"},
    {`(/ 12 2 0)`, "ERR:/: Division durch 0"},
    {`(/ 12 0 2)`, "ERR:/: Division durch 0"},
    {`(/ 12 2 0.0)`, "ERR:/: Division durch 0"},
    {`(/ ` + minI + ` 1 -1)`, "ERR:/: Ganzzahl-Überlauf"},
    {`(/ 5)`, "0.2"},
    {`(/ 1)`, "1"},
    {`(/ 0.5)`, "2.0"},
    {`(/ 0)`, "ERR:/: Division durch 0"},
  })
}
