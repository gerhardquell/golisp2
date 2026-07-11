//      lib/shm/shm.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//########################################

package shm

import (
  "syscall"
  "unsafe"
)

// ------------------------------------------------------------------------------
const (
  IPC_CREAT = 01000
  IPC_EXCL = 02000
  IPC_NOWAIT = 04000
  IPC_PRIVATE = 0
  SHM_RDONLY = 010000
  SHM_RND = 020000
  SHM_REMAP = 040000
  SHM_EXEC = 0100000
  SHM_LOCK = 1
  SHM_UNLOCK = 12
  IPC_RMID = 0
  IPC_SET = 1
  IPC_STAT = 2
  sysShmAt  = syscall.SYS_SHMAT
  sysShmCtl = syscall.SYS_SHMCTL
  sysShmDt  = syscall.SYS_SHMDT
  sysShmGet = syscall.SYS_SHMGET
)

// ------------------------------------------------------------------------------
// ##############################################################################
func ShmGet(key int, size int, shmFlg int) (shmId int, err error) {
  id, _, errno := syscall.Syscall(sysShmGet, uintptr(int32(key)),
                                  uintptr(int32(size)), uintptr(int32(shmFlg)))
  if int(id) == -1 { return -1, errno }
  return int(id), nil
}

// ##############################################################################
func ShmAt(shmId int, shmAddr uintptr, shmFlg int, size int) (data []byte, err error) {
  addr, _, errno := syscall.Syscall(sysShmAt, uintptr(int32(shmId)), shmAddr,
                                    uintptr(int32(shmFlg)))
  if int(addr) == -1 { return nil, errno }
  var b = struct {
    addr uintptr
    len  int
    cap  int
  }{addr, size, size}
  data = *(*[]byte)(unsafe.Pointer(&b))
  return data, nil
}

// ##############################################################################
func ShmDt(data []byte) error {
  result, _, errno := syscall.Syscall(sysShmDt,
                                      uintptr(unsafe.Pointer(&data[0])),  0, 0)
  if int(result) == -1 { return errno }
  return nil
}

// ##############################################################################
func ShmRm(shmId int) error {
  result, _, errno := syscall.Syscall(sysShmCtl, uintptr(int32(shmId)),
                                      uintptr(int32(IPC_RMID)), 0)
  if int(result) == -1 { return errno }
  return nil
}
// ##############################################################################
