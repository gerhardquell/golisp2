//**********************************************************************
//  lib/eval_exec.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************
// Spezialform: (exec "prog" param: "arg" ... stdout: var stderr: var
//               exitcd: var stdin: input)
//**********************************************************************

package lib

import (
  "bytes"
  "context"
  "fmt"
  "os/exec"
  "strings"
  "time"
)

const defaultExecTimeout = 60 * time.Second

func evalExec(args *Cell, env *Env) (*Cell, error) {
  if args == nil || args.Car == nil {
    return nil, fmt.Errorf("exec: Programmname erwartet")
  }

  programCell, err := Eval(args.Car, env)
  if err != nil {
    return nil, fmt.Errorf("exec: %v", err)
  }
  if programCell == nil || programCell.Type != STRING {
    return nil, fmt.Errorf("exec: Programmname muss ein String sein")
  }
  program := programCell.Val

  var params []string
  var stdinStr string
  var stdoutVar, stderrVar, exitcdVar string

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
      val, err := Eval(valueCell, env)
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val == nil || val.Type != STRING {
        return nil, fmt.Errorf("exec: param muss String sein")
      }
      params = append(params, val.Val)
    case "stdin:":
      val, err := Eval(valueCell, env)
      if err != nil {
        return nil, fmt.Errorf("exec: %v", err)
      }
      if val != nil && val.Type != STRING {
        return nil, fmt.Errorf("exec: stdin muss String sein")
      }
      if val != nil {
        stdinStr = val.Val
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

  ctx, cancel := context.WithTimeout(context.Background(), defaultExecTimeout)
  defer cancel()

  cmd := exec.CommandContext(ctx, program, params...)
  var stdoutBuf, stderrBuf bytes.Buffer
  cmd.Stdout = &stdoutBuf
  cmd.Stderr = &stderrBuf
  cmd.Stdin = strings.NewReader(stdinStr)

  exitCode := 0
  runErr := cmd.Run()

  if runErr != nil {
    if exitErr, ok := runErr.(*exec.ExitError); ok {
      exitCode = exitErr.ExitCode()
    } else {
      if exitcdVar != "" {
        env.Set(exitcdVar, MakeNum(-1))
      }
      return MakeNil(), nil
    }
  }

  if stdoutVar != "" {
    env.Set(stdoutVar, MakeStr(stdoutBuf.String()))
  }
  if stderrVar != "" {
    env.Set(stderrVar, MakeStr(stderrBuf.String()))
  }
  if exitcdVar != "" {
    env.Set(exitcdVar, MakeNum(float64(exitCode)))
  }

  return cellT, nil
}
