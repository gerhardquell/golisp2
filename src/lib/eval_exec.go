//**********************************************************************
//  lib/eval_exec.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************
// Spezialform: (exec "prog" param: "arg" ... env: "KEY=WERT" ...
//               stdout: var stderr: var exitcd: var stdin: input
//               timeout: N)
// param: und env: koennen mehrfach angegeben werden. timeout: N (Sekunden,
// NUMBER) ueberschreibt defaultExecTimeout; N = -1 bedeutet kein Timeout
// (context.Background() ohne Deadline). N = 0 oder N < -1 ist ein Fehler.
//**********************************************************************

package lib

import (
  "bytes"
  "context"
  "fmt"
  "os"
  "os/exec"
  "strings"
  "time"
)

const defaultExecTimeout = 60 * time.Second

func evalExec(args *Cell, env *Env, ectx evalCtx) (*Cell, error) {
  if args == nil || args.Car == nil {
    return nil, fmt.Errorf("exec: Programmname erwartet")
  }

  programCell, err := evalWithCtx(args.Car, env, ectx.child())
  if err != nil {
    return nil, fmt.Errorf("exec: %v", err)
  }
  if programCell == nil || programCell.Type != STRING {
    return nil, fmt.Errorf("exec: Programmname muss ein String sein")
  }
  program := programCell.Val

  var params []string
  var envVars []string
  var stdinStr string
  var stdoutVar, stderrVar, exitcdVar string
  timeout := defaultExecTimeout
  noTimeout := false

  rest := args.Cdr
  for rest != nil && rest.Type == LIST {
    if rest.Car == nil {
      break
    }
    keywordCell := rest.Car
    if keywordCell.Type != ATOM {
      return nil, fmt.Errorf("exec: Keyword erwartet, erhalten: %v", keywordCell)
    }
    keyword := keywordCell.Val

    if rest.Cdr == nil || rest.Cdr.Car == nil {
      return nil, fmt.Errorf("exec: Wert fuer %s fehlt", keyword)
    }
    valueCell := rest.Cdr.Car

    switch keyword {
    case "param:":
      val, err := evalWithCtx(valueCell, env, ectx.child())
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val == nil || val.Type != STRING {
        return nil, fmt.Errorf("exec: param muss String sein")
      }
      params = append(params, val.Val)
    case "env:":
      val, err := evalWithCtx(valueCell, env, ectx.child())
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val == nil || val.Type != STRING {
        return nil, fmt.Errorf("exec: env muss String der Form KEY=WERT sein")
      }
      if !strings.Contains(val.Val, "=") {
        return nil, fmt.Errorf("exec: env erwartet Form KEY=WERT, erhalten: %s", val.Val)
      }
      envVars = append(envVars, val.Val)
    case "stdin:":
      val, err := evalWithCtx(valueCell, env, ectx.child())
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val != nil && val.Type != STRING {
        return nil, fmt.Errorf("exec: stdin muss String sein")
      }
      if val != nil {
        stdinStr = val.Val
      }
    case "timeout:":
      val, err := evalWithCtx(valueCell, env, ectx.child())
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val == nil || val.Type != NUMBER {
        return nil, fmt.Errorf("exec: timeout muss NUMBER sein")
      }
      if val.Num == -1 {
        noTimeout = true
      } else if val.Num <= 0 {
        return nil, fmt.Errorf("exec: timeout muss > 0 oder -1 (unendlich) sein")
      } else {
        timeout = time.Duration(val.Num * float64(time.Second))
      }
    case "stdout:", "stderr:", "exitcd:":
      if valueCell.Type != ATOM {
        return nil, fmt.Errorf("exec: %s erwartet Variablennamen", keyword)
      }
      name := valueCell.Val
      switch keyword {
      case "stdout:":
        stdoutVar = name
      case "stderr:":
        stderrVar = name
      case "exitcd:":
        exitcdVar = name
      }
    default:
      return nil, fmt.Errorf("exec: unbekanntes Keyword %s", keyword)
    }

    rest = rest.Cdr.Cdr
  }

  var ctx context.Context
  var cancel context.CancelFunc
  if noTimeout {
    ctx, cancel = context.WithCancel(context.Background())
  } else {
    ctx, cancel = context.WithTimeout(context.Background(), timeout)
  }
  defer cancel()

  cmd := exec.CommandContext(ctx, program, params...)
  if len(envVars) > 0 {
    cmd.Env = append(os.Environ(), envVars...)
  }
  var stdoutBuf, stderrBuf bytes.Buffer
  cmd.Stdout = &stdoutBuf
  cmd.Stderr = &stderrBuf
  cmd.Stdin = strings.NewReader(stdinStr)

  exitCode := 0
  runErr := cmd.Run()

  if ctx.Err() != nil {
    if exitcdVar != "" {
      if err := env.Set(exitcdVar, MakeNum(-1)); err != nil { return nil, err }
    }
    return MakeNil(), nil
  }

  if runErr != nil {
    if exitErr, ok := runErr.(*exec.ExitError); ok {
      exitCode = exitErr.ExitCode()
    } else {
      if exitcdVar != "" {
        if err := env.Set(exitcdVar, MakeNum(-1)); err != nil { return nil, err }
      }
      return MakeNil(), nil
    }
  }

  if stdoutVar != "" {
    if err := env.Set(stdoutVar, MakeStr(stdoutBuf.String())); err != nil { return nil, err }
  }
  if stderrVar != "" {
    if err := env.Set(stderrVar, MakeStr(stderrBuf.String())); err != nil { return nil, err }
  }
  if exitcdVar != "" {
    if err := env.Set(exitcdVar, MakeNum(float64(exitCode))); err != nil { return nil, err }
  }

  return cellT, nil
}
