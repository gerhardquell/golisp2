//**********************************************************************
//  lib/format_dirs.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260625
//**********************************************************************
// FORMAT-Direktiven-Handler + Helper (aufgeteilt aus format.go).
// Siehe format.go für Engine und Semantik.
//**********************************************************************

package lib

import (
  "fmt"
  "strconv"
  "strings"
  "unicode/utf8"
)

// ---- Ausgabe-Formen ----

// aesthetic: ~A — ohne Quotes (princ-Stil).
func aesthetic(c *Cell) string {
  if c == nil {
    return "()"
  }
  switch c.Type {
  case NIL:
    return "()"
  case STRING:
    return c.Val
  case ATOM:
    return c.Val
  case NUMBER:
    return numStr(c.Num)
  default:
    return c.String()
  }
}

// readable: ~S — mit Quotes (prin1-Stil).
func readable(c *Cell) string {
  if c == nil {
    return "()"
  }
  return c.String()
}

func numStr(n float64) string {
  if n == float64(int64(n)) {
    return strconv.FormatInt(int64(n), 10)
  }
  return strconv.FormatFloat(n, 'g', -1, 64)
}

// ---- Padding ----

func writeN(out *strings.Builder, r rune, n int) {
  if n <= 0 {
    return
  }
  for i := 0; i < n; i++ {
    out.WriteRune(r)
  }
}

func freshLine(out *strings.Builder, n int) {
  s := out.String()
  if s == "" || s[len(s)-1] != '\n' {
    out.WriteRune('\n')
    n--
  }
  writeN(out, '\n', n)
}

// emitPadded: ~A/~S mit mincol,colinc,minpad,padchar. colon=rechtsbündig.
func emitPadded(st *fmtState, s string, params []pval, colon bool) {
  mincol := resolveInt(st, paramGet(params, 0), 0)
  colinc := resolveInt(st, paramGet(params, 1), 1)
  minpad := resolveInt(st, paramGet(params, 2), 0)
  padchar := resolveRune(st, paramGet(params, 3), ' ')
  if colinc < 1 {
    colinc = 1
  }
  st.out.WriteString(padField(s, mincol, colinc, minpad, padchar, colon))
}

func padField(s string, mincol, colinc, minpad int, padchar rune, rightAlign bool) string {
  w := utf8.RuneCountInString(s)
  if w >= mincol && minpad <= 0 {
    return s
  }
  pad := minpad
  if w+pad < mincol {
    need := mincol - w
    if colinc > 0 {
      extra := need - pad
      if extra > 0 {
        pad += ((extra + colinc - 1) / colinc) * colinc
      }
    } else {
      pad = need
    }
  }
  if pad <= 0 {
    return s
  }
  fill := strings.Repeat(string(padchar), pad)
  if rightAlign {
    return fill + s
  }
  return s + fill
}

// ---- Radix-Direktiven ~D ~B ~O ~X ----

func emitRadix(st *fmtState, params []pval, colon, at bool, base int) {
  c := st.consume()
  n := toInt(c)
  mincol := resolveInt(st, paramGet(params, 0), 0)
  padchar := resolveRune(st, paramGet(params, 1), ' ')
  commachar := resolveRune(st, paramGet(params, 2), ',')
  commainterval := resolveInt(st, paramGet(params, 3), 3)
  if commainterval < 1 {
    commainterval = 3
  }
  neg := n < 0
  abs := n
  if neg {
    abs = -n
  }
  digits := strconv.FormatInt(int64(abs), base)
  if colon {
    digits = insertCommas(digits, commachar, commainterval)
  }
  sign := ""
  if neg {
    sign = "-"
  } else if at {
    sign = "+"
  }
  s := sign + digits
  if mincol > utf8.RuneCountInString(s) {
    pad := strings.Repeat(string(padchar), mincol-utf8.RuneCountInString(s))
    s = pad + s
  }
  st.out.WriteString(s)
}

func insertCommas(digits string, sep rune, interval int) string {
  r := []rune(digits)
  var b strings.Builder
  first := len(r) % interval
  if first == 0 && len(r) > 0 {
    first = interval
  }
  for i, ch := range r {
    if i > 0 && (i-first)%interval == 0 && i <= len(r) {
      b.WriteRune(sep)
    }
    b.WriteRune(ch)
  }
  return b.String()
}

func toInt(c *Cell) int64 {
  if c == nil {
    return 0
  }
  if c.Type == NUMBER {
    return int64(c.Num)
  }
  if c.Type == STRING {
    n, _ := strconv.ParseInt(c.Val, 10, 64)
    return n
  }
  return 0
}

// ---- ~R ----

func emitR(st *fmtState, params []pval, colon, at bool) {
  c := st.consume()
  n := toInt(c)
  baseParam := paramGet(params, 0)
  if baseParam.kind != pMissing {
    base := resolveInt(st, baseParam, 10)
    if base < 2 || base > 36 {
      st.err = fmt.Errorf("format: ~R base %d außerhalb 2..36", base)
      return
    }
    neg := n < 0
    abs := n
    if neg {
      abs = -n
    }
    s := strconv.FormatInt(int64(abs), base)
    if neg {
      s = "-" + s
    }
    st.out.WriteString(s)
    return
  }
  if colon && at {
    // old roman
    st.out.WriteString(oldRoman(int(n)))
    return
  }
  if at {
    st.out.WriteString(roman(int(n)))
    return
  }
  if colon {
    st.out.WriteString(ordinal(int(n)))
    return
  }
  st.out.WriteString(cardinal(int(n)))
}

func cardinal(n int) string {
  if n == 0 {
    return "zero"
  }
  neg := n < 0
  if neg {
    n = -n
  }
  s := cardinalPos(n)
  if neg {
    s = "minus " + s
  }
  return s
}

func cardinalPos(n int) string {
  if n < 0 {
    return ""
  }
  ones := []string{"zero", "one", "two", "three", "four", "five", "six",
    "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen",
    "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
  tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty",
    "seventy", "eighty", "ninety"}
  switch {
  case n < 20:
    return ones[n]
  case n < 100:
    t := tens[n/10]
    if n%10 != 0 {
      return t + "-" + ones[n%10]
    }
    return t
  case n < 1000:
    s := ones[n/100] + " hundred"
    if n%100 != 0 {
      return s + " " + cardinalPos(n%100)
    }
    return s
  case n < 1000000:
    s := cardinalPos(n/1000) + " thousand"
    if n%1000 != 0 {
      return s + " " + cardinalPos(n%1000)
    }
    return s
  }
  return strconv.Itoa(n)
}

func ordinal(n int) string {
  if n == 0 {
    return "zeroth"
  }
  neg := n < 0
  a := n
  if neg {
    a = -n
  }
  s := ordinalPos(a)
  if neg {
    s = "minus " + s
  }
  return s
}

func ordinalPos(n int) string {
  ones := []string{"zeroth", "first", "second", "third", "fourth", "fifth",
    "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelfth",
    "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth",
    "eighteenth", "nineteenth"}
  tens := []string{"", "", "twentieth", "thirtieth", "fortieth", "fiftieth",
    "sixtieth", "seventieth", "eightieth", "ninetieth"}
  switch {
  case n < 20:
    return ones[n]
  case n < 100:
    if n%10 == 0 {
      return tens[n/10]
    }
    return cardinalPos(n-n%10) + "-" + ordinalPos(n%10)
  case n < 1000:
    if n%100 == 0 {
      return cardinalPos(n/100) + " hundredth"
    }
    return cardinalPos(n-n%100) + " " + ordinalPos(n%100)
  case n < 1000000:
    if n%1000 == 0 {
      return cardinalPos(n/1000) + " thousandth"
    }
    return cardinalPos(n-n%1000) + " " + ordinalPos(n%1000)
  }
  return strconv.Itoa(n) + "th"
}

func roman(n int) string {
  if n < 1 || n > 3999 {
    return strconv.Itoa(n)
  }
  vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
  syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
  var b strings.Builder
  for i, v := range vals {
    for n >= v {
      b.WriteString(syms[i])
      n -= v
    }
  }
  return b.String()
}

func oldRoman(n int) string {
  if n < 1 || n > 3999 {
    return strconv.Itoa(n)
  }
  var b strings.Builder
  writeN(&b, 'M', n/1000)
  n %= 1000
  writeN(&b, 'D', n/500)
  n %= 500
  writeN(&b, 'C', n/100)
  n %= 100
  writeN(&b, 'L', n/50)
  n %= 50
  writeN(&b, 'X', n/10)
  n %= 10
  writeN(&b, 'V', n/5)
  n %= 5
  writeN(&b, 'I', n)
  return b.String()
}

// ---- ~P ----

func emitP(st *fmtState, colon, at bool) {
  c := st.consume()
  n := toInt(c)
  one := n == 1
  if colon {
    if one {
      st.out.WriteString("y")
    } else {
      st.out.WriteString("ies")
    }
    return
  }
  // at: Argument nicht verbrauchen → zurückstellen
  if at {
    st.idx--
    if one {
      st.out.WriteString("y")
    } else {
      st.out.WriteString("ies")
    }
    return
  }
  if !one {
    st.out.WriteString("s")
  }
}

// ---- ~C ----

func emitC(st *fmtState, params []pval, colon, at bool) {
  c := st.consume()
  var r rune
  switch {
  case c.Type == STRING && utf8.RuneCountInString(c.Val) >= 1:
    r, _ = utf8.DecodeRuneInString(c.Val)
  case c.Type == NUMBER:
    r = rune(int(c.Num))
  default:
    st.err = fmt.Errorf("format: ~C braucht Zeichen (String oder Zahl)")
    return
  }
  if colon || at {
    if name, ok := charName(r); ok {
      st.out.WriteString(name)
    } else {
      st.out.WriteRune(r)
    }
    return
  }
  st.out.WriteRune(r)
}

func charName(r rune) (string, bool) {
  names := map[rune]string{
    ' ': "Space", '\n': "Newline", '\t': "Tab", '\r': "Return",
    '\f': "Page", '\v': "Vtab", 0: "Null", 127: "Rubout",
  }
  if n, ok := names[r]; ok {
    return n, true
  }
  if r < 32 || r == 127 {
    return "", true // kein Name → Zeichen selbst
  }
  return "", false
}

// ---- Float ~F ~E ~G ~$ ----

func emitF(st *fmtState, params []pval, at bool) {
  w := resolveInt(st, paramGet(params, 0), 0)
  // d: fehlt → volle Precision (-1); 0 explizit → kein Nachkomma
  d := -1
  if dp := paramGet(params, 1); dp.kind != pMissing {
    d = resolveInt(st, dp, 0)
  }
  k := resolveInt(st, paramGet(params, 2), 0) // Skalierung ×10^k
  // overflowchar (param[3]) nicht unterstützt — dokumentiert.
  padchar := resolveRune(st, paramGet(params, 4), ' ')
  c := st.consume()
  f := toFloat(c) * pow10(k)
  s := formatFixed(f, d, at)
  if w > utf8.RuneCountInString(s) {
    s = strings.Repeat(string(padchar), w-utf8.RuneCountInString(s)) + s
  }
  st.out.WriteString(s)
}

// pow10: 10^k ohne math-Import (k kann negativ sein).
func pow10(k int) float64 {
  r := 1.0
  if k >= 0 {
    for i := 0; i < k; i++ {
      r *= 10
    }
  } else {
    for i := 0; i < -k; i++ {
      r /= 10
    }
  }
  return r
}

func formatFixed(f float64, d int, at bool) string {
  neg := f < 0
  if neg {
    f = -f
  }
  sign := ""
  if neg {
    sign = "-"
  } else if at {
    sign = "+"
  }
  if d < 0 {
    // volle Precision (minimal darstellbar)
    return sign + strconv.FormatFloat(f, 'g', -1, 64)
  }
  return sign + strconv.FormatFloat(f, 'f', d, 64)
}

func emitE(st *fmtState, params []pval, at bool) {
  w := resolveInt(st, paramGet(params, 0), 0)
  d := resolveInt(st, paramGet(params, 1), 0)
  e := resolveInt(st, paramGet(params, 2), 2)
  padchar := resolveRune(st, paramGet(params, 5), ' ')
  c := st.consume()
  f := toFloat(c)
  s := formatExp(f, d, e, at)
  if w > utf8.RuneCountInString(s) {
    s = strings.Repeat(string(padchar), w-utf8.RuneCountInString(s)) + s
  }
  st.out.WriteString(s)
}

func formatExp(f float64, d, e int, at bool) string {
  if d < 0 {
    d = 0
  }
  neg := f < 0
  if neg {
    f = -f
  }
  s := strconv.FormatFloat(f, 'e', d, 64)
  // Normalisiere Exponenten-Breite
  parts := strings.SplitN(s, "e", 2)
  mant := parts[0]
  exp := 0
  if len(parts) == 2 {
    exp, _ = strconv.Atoi(parts[1])
  }
  sign := ""
  if neg {
    sign = "-"
  } else if at {
    sign = "+"
  }
  esign := "+"
  if exp < 0 {
    esign = "-"
    exp = -exp
  }
  es := strconv.Itoa(exp)
  if e > len(es) {
    es = strings.Repeat("0", e-len(es)) + es
  }
  return sign + mant + "e" + esign + es
}

func emitG(st *fmtState, params []pval, at bool) {
  // ~G: General — vereinfacht als ~E-Darstellung gewählt.
  emitE(st, params, at)
}

func emitDollar(st *fmtState, params []pval, at bool) {
  d := resolveInt(st, paramGet(params, 0), 2)
  n := resolveInt(st, paramGet(params, 1), 0) // min Ziffern vor Punkt
  w := resolveInt(st, paramGet(params, 2), 0)
  padchar := resolveRune(st, paramGet(params, 3), ' ')
  c := st.consume()
  f := toFloat(c)
  neg := f < 0
  if neg {
    f = -f
  }
  s := strconv.FormatFloat(f, 'f', d, 64)
  // Ganzzahl-Teil vor Punkt auf mindestenst n Ziffern paden
  if dot := strings.IndexByte(s, '.'); dot >= 0 {
    intPart := s[:dot]
    if len(intPart) < n {
      intPart = strings.Repeat(string(padchar), n-len(intPart)) + intPart
    }
    s = intPart + s[dot:]
  }
  sign := ""
  if neg {
    sign = "-"
  } else if at {
    sign = "+"
  }
  body := sign + s
  if w > utf8.RuneCountInString(body) {
    body = strings.Repeat(string(padchar), w-utf8.RuneCountInString(body)) + body
  }
  st.out.WriteString(body)
}

func toFloat(c *Cell) float64 {
  if c == nil {
    return 0
  }
  if c.Type == NUMBER {
    return c.Num
  }
  if c.Type == STRING {
    f, _ := strconv.ParseFloat(c.Val, 64)
    return f
  }
  return 0
}

// ---- ~T Tabulate ----

func emitT(st *fmtState, params []pval, colon, at bool) {
  colnum := resolveInt(st, paramGet(params, 0), 1)
  colinc := resolveInt(st, paramGet(params, 1), 1)
  cur := utf8.RuneCountInString(st.out.String())
  if at {
    // relativ zur aktuellen Position
    writeN(st.out, ' ', colnum)
    if colinc > 0 {
      for (cur+colnum)%colinc != 0 {
        st.out.WriteRune(' ')
        cur++
      }
    }
    return
  }
  if colon {
    // relativ, rückwärts-orientiert (vereinfacht wie at)
    writeN(st.out, ' ', colnum)
    return
  }
  // absolut: Springe zu Spalte colnum
  if cur < colnum {
    writeN(st.out, ' ', colnum-cur)
  } else if colinc > 0 {
    for cur < colnum || (cur-colnum)%colinc != 0 {
      st.out.WriteRune(' ')
      cur++
    }
  }
}

// ---- ~* Goto ----

func emitGoto(st *fmtState, params []pval, colon, at bool) {
  n := resolveInt(st, paramGet(params, 0), 1)
  if at {
    st.idx = n
  } else if colon {
    st.idx -= n
    if st.idx < 0 {
      st.idx = 0
    }
  } else {
    st.idx += n
  }
}

