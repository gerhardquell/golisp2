//**********************************************************************
//  lib/shm_lisp.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//**********************************************************************
// Shared-Memory Primitiven fuer GoLisp2 (High-Level Pool-API)
//**********************************************************************

package lib

import (
  "fmt"
  "golisp2/lib/shm"
)

// shmHandle verpackt einen ShmBlock als opake Lisp-Ressource
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
  if c == nil || c.Type != FUNC || c.Val != "shm-block" {
    return nil, fmt.Errorf("shm-block: kein SHM-Block")
  }
  h, ok := c.Env.(*shmHandle)
  if !ok {
    return nil, fmt.Errorf("shm-block: kein SHM-Block")
  }
  return h.block, nil
}

func RegisterShm(env *Env) {
  env.Set("shm-alloc",   makeFn(fnShmAlloc))
  env.Set("shm-free",    makeFn(fnShmFree))
  env.Set("shm-write",   makeFn(fnShmWrite))
  env.Set("shm-read",    makeFn(fnShmRead))
  env.Set("shm-status",  makeFn(fnShmStatus))
  env.Set("shm-cleanup", makeFn(fnShmCleanup))
}

func fnShmAlloc(args []*Cell) (*Cell, error) {
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-alloc: %v", err)
  }
  block, err := pool.Allocate(0)
  if err != nil {
    return nil, fmt.Errorf("shm-alloc: %v", err)
  }
  return makeShmCell(block), nil
}

func fnShmFree(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("shm-free: 1 Argument nötig")
  }
  block, err := getShmBlock(args[0])
  if err != nil {
    return nil, fmt.Errorf("shm-free: %v", err)
  }
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-free: %v", err)
  }
  if err := pool.Release(block.ID); err != nil {
    return nil, fmt.Errorf("shm-free: %v", err)
  }
  return cellT, nil
}

func fnShmWrite(args []*Cell) (*Cell, error) {
  if len(args) < 2 {
    return nil, fmt.Errorf("shm-write: 2 Argumente nötig")
  }
  if args[1].Type != STRING {
    return nil, fmt.Errorf("shm-write: String erwartet")
  }
  block, err := getShmBlock(args[0])
  if err != nil {
    return nil, fmt.Errorf("shm-write: %v", err)
  }
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-write: %v", err)
  }
  if err := pool.WriteData(block.ID, []byte(args[1].Val)); err != nil {
    return nil, fmt.Errorf("shm-write: %v", err)
  }
  return args[1], nil
}

func fnShmRead(args []*Cell) (*Cell, error) {
  if len(args) < 1 {
    return nil, fmt.Errorf("shm-read: 1 Argument nötig")
  }
  block, err := getShmBlock(args[0])
  if err != nil {
    return nil, fmt.Errorf("shm-read: %v", err)
  }

  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-read: %v", err)
  }
  data, err := pool.ReadData(block.ID, shm.POOL_SIZE)
  if err != nil {
    return nil, fmt.Errorf("shm-read: %v", err)
  }

  n := len(data)
  if len(args) > 1 {
    if args[1].Type != NUMBER {
      return nil, fmt.Errorf("shm-read: Zahl erwartet")
    }
    want := int(args[1].Num)
    if want < 0 { want = 0 }
    if want < n { n = want }
  } else {
    for i, b := range data {
      if b == 0 {
        n = i
        break
      }
    }
  }

  return MakeStr(string(data[:n])), nil
}

func fnShmStatus(args []*Cell) (*Cell, error) {
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-status: %v", err)
  }
  pool.Status()
  return cellT, nil
}

func fnShmCleanup(args []*Cell) (*Cell, error) {
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-cleanup: %v", err)
  }
  pool.Cleanup()
  return cellT, nil
}
