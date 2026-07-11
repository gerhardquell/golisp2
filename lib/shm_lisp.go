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
  if c == nil || c.Val != "shm-block" {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  h, ok := c.Env.(*shmHandle)
  if !ok {
    return nil, fmt.Errorf("kein SHM-Block")
  }
  return h.block, nil
}

func RegisterShm(env *Env) {
  env.Set("shm-alloc", makeFn(fnShmAlloc))
  env.Set("shm-free",  makeFn(fnShmFree))
}

func fnShmAlloc(args []*Cell) (*Cell, error) {
  block, err := shm.GetPool().Allocate(0)
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
  if err := shm.GetPool().Release(block.ID); err != nil {
    return nil, fmt.Errorf("shm-free: %v", err)
  }
  return cellT, nil
}
