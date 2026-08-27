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
  "golisp2/src/lib/shm"
)

// shmHandle verpackt einen ShmBlock als opake Lisp-Ressource.
// Zusätzlich zur Pool-Referenz wird die Block-ID gespeichert, damit Handles
// nach einem shm-cleanup (Pool-Reset) invalidiert werden können.
type shmHandle struct {
  pool     *shm.ShmPool
  blockID  int
}

func makeShmCell(pool *shm.ShmPool, block *shm.ShmBlock) *Cell {
  return &Cell{
    Type: FUNC,
    Val:  "shm-block",
    Fn: func(_ []*Cell) (*Cell, error) {
      return nil, fmt.Errorf("shm-block: Handle ist nicht aufrufbar")
    },
    Env:  &shmHandle{pool: pool, blockID: block.ID},
  }
}

func getShmBlock(c *Cell) (*shm.ShmBlock, error) {
  if c == nil || c.Type != FUNC || c.Val != "shm-block" {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  h, ok := c.Env.(*shmHandle)
  if !ok || h.pool == nil {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  if h.pool != pool {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  return pool.GetBlock(h.blockID)
}

func RegisterShm(env *Env) {
  _ = env.Set("shm-alloc",   makeFn(fnShmAlloc))
  _ = env.Set("shm-free",    makeFn(fnShmFree))
  _ = env.Set("shm-write",   makeFn(fnShmWrite))
  _ = env.Set("shm-read",    makeFn(fnShmRead))
  _ = env.Set("shm-status",  makeFn(fnShmStatus))
  _ = env.Set("shm-cleanup", makeFn(fnShmCleanup))
}

func fnShmAlloc(args []*Cell) (*Cell, error) {
  workerID := 0
  if len(args) == 1 {
    if args[0].Type != NUMBER {
      return nil, fmt.Errorf("shm-alloc: optionale Worker-ID muss eine Zahl sein")
    }
    workerID = int(args[0].Num)
  } else if len(args) > 1 {
    return nil, fmt.Errorf("shm-alloc: 0 oder 1 Argumente erwartet")
  }

  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-alloc: %v", err)
  }
  block, err := pool.Allocate(workerID)
  if err != nil {
    return nil, fmt.Errorf("shm-alloc: %v", err)
  }
  return makeShmCell(pool, block), nil
}

func fnShmFree(args []*Cell) (*Cell, error) {
  if len(args) != 1 {
    return nil, fmt.Errorf("shm-free: genau 1 Argument erwartet")
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
  if len(args) != 2 {
    return nil, fmt.Errorf("shm-write: genau 2 Argumente erwartet")
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
  if len(args) < 1 || len(args) > 2 {
    return nil, fmt.Errorf("shm-read: 1 oder 2 Argumente erwartet")
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
  if len(args) != 0 {
    return nil, fmt.Errorf("shm-status: keine Argumente erlaubt")
  }
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-status: %v", err)
  }
  status, err := pool.Status()
  if err != nil {
    return nil, fmt.Errorf("shm-status: %v", err)
  }

  entries := []*Cell{
    Cons(MakeAtom("total"), MakeNum(float64(status.Total))),
    Cons(MakeAtom("used"),  MakeNum(float64(status.Used))),
    Cons(MakeAtom("free"),  MakeNum(float64(status.Free))),
  }
  return SliceToCell(entries), nil
}

func fnShmCleanup(args []*Cell) (*Cell, error) {
  if len(args) != 0 {
    return nil, fmt.Errorf("shm-cleanup: keine Argumente erlaubt")
  }
  pool, err := shm.GetPool()
  if err != nil {
    return nil, fmt.Errorf("shm-cleanup: %v", err)
  }
  freed, err := pool.Cleanup()
  if err != nil {
    return nil, fmt.Errorf("shm-cleanup: %v", err)
  }
  return MakeNum(float64(freed)), nil
}
