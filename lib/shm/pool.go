//      lib/shm/pool.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//########################################

package shm

import (
  "errors"
  "fmt"
  "sync"
)

//########################################
const (
  MAX_POOLS = 150
  POOL_SIZE = 2048 * 1024  // 2MB pro Pool
  SHM_BASE  = 0x2000       // Basis SHM-Key
)

//########################################
type ShmBlock struct {
  ID       int
  Key      int
  ShmID    int
  Size     int
  Data     []byte
  InUse    bool
  WorkerID int
}

//########################################
type ShmBlockStatus struct {
  ID       int
  Key      int
  ShmID    int
  InUse    bool
  WorkerID int
}

//########################################
type ShmPoolStatus struct {
  Total  int
  Used   int
  Free   int
  Blocks []ShmBlockStatus
}

//########################################
type ShmPool struct {
  blocks [MAX_POOLS]*ShmBlock
  mutex  sync.Mutex
  inited bool
}

//########################################
var globalPool *ShmPool
var poolMutex sync.Mutex

//########################################
func GetPool() (*ShmPool, error) {
  poolMutex.Lock()
  defer poolMutex.Unlock()

  if globalPool != nil && globalPool.inited {
    return globalPool, nil
  }

  p := &ShmPool{}
  if err := p.Init(); err != nil {
    return nil, fmt.Errorf("GetPool: %v", err)
  }
  globalPool = p
  return p, nil
}

//########################################
func (p *ShmPool) Init() error {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if p.inited {
    return nil
  }

  created := make([]*ShmBlock, 0, MAX_POOLS)

  for i := range MAX_POOLS {
    key := SHM_BASE + i

    shmID, err := ShmGet(key, POOL_SIZE, IPC_CREAT|0600)
    if err != nil {
      p.rollback(created)
      return fmt.Errorf("Init: SHM-Segment für Pool %d konnte nicht erzeugt werden: %v", i, err)
    }

    data, err := ShmAt(shmID, 0, 0, POOL_SIZE)
    if err != nil {
      ShmRm(shmID)
      p.rollback(created)
      return fmt.Errorf("Init: SHM-Segment für Pool %d konnte nicht angehängt werden: %v", i, err)
    }

    block := &ShmBlock{
      ID:       i,
      Key:      key,
      ShmID:    shmID,
      Size:     POOL_SIZE,
      Data:     data,
      InUse:    false,
      WorkerID: -1,
    }
    p.blocks[i] = block
    created = append(created, block)
  }

  p.inited = true
  return nil
}

//########################################
func (p *ShmPool) rollback(created []*ShmBlock) {
  for _, block := range created {
    if block == nil {
      continue
    }
    if block.Data != nil {
      ShmDt(block.Data)
    }
    if block.ShmID != 0 {
      ShmRm(block.ShmID)
    }
  }
}

//########################################
func (p *ShmPool) Allocate(workerID int) (*ShmBlock, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if !p.inited {
    return nil, fmt.Errorf("Allocate: Pool ist nicht initialisiert")
  }

  for i := range MAX_POOLS {
    if !p.blocks[i].InUse {
      p.blocks[i].InUse = true
      p.blocks[i].WorkerID = workerID
      return p.blocks[i], nil
    }
  }

  return nil, fmt.Errorf("Allocate: kein freier SHM-Block verfügbar")
}

//########################################
func (p *ShmPool) Release(blockID int) error {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if !p.inited {
    return fmt.Errorf("Release: Pool ist nicht initialisiert")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("Release: ungültige Block-ID %d", blockID)
  }

  if !p.blocks[blockID].InUse {
    return fmt.Errorf("Release: Block %d ist nicht alloziert", blockID)
  }

  p.blocks[blockID].InUse = false
  p.blocks[blockID].WorkerID = -1

  for i := range p.blocks[blockID].Data {
    p.blocks[blockID].Data[i] = 0
  }

  return nil
}

//########################################
func (p *ShmPool) WriteData(blockID int, data []byte) error {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if !p.inited {
    return fmt.Errorf("WriteData: Pool ist nicht initialisiert")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("WriteData: ungültige Block-ID %d", blockID)
  }

  block := p.blocks[blockID]
  if !block.InUse {
    return fmt.Errorf("WriteData: Block %d ist nicht alloziert", blockID)
  }

  if len(data) > block.Size {
    return fmt.Errorf("WriteData: Daten zu groß: %d > %d", len(data), block.Size)
  }

  copy(block.Data, data)
  for i := len(data); i < len(block.Data); i++ {
    block.Data[i] = 0
  }
  return nil
}

//########################################
func (p *ShmPool) ReadData(blockID int, maxLen int) ([]byte, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if !p.inited {
    return nil, fmt.Errorf("ReadData: Pool ist nicht initialisiert")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return nil, fmt.Errorf("ReadData: ungültige Block-ID %d", blockID)
  }

  block := p.blocks[blockID]
  if !block.InUse {
    return nil, fmt.Errorf("ReadData: Block %d ist nicht alloziert", blockID)
  }

  if maxLen > block.Size {
    maxLen = block.Size
  }

  result := make([]byte, maxLen)
  copy(result, block.Data[:maxLen])
  return result, nil
}

//########################################
func (p *ShmPool) Status() (*ShmPoolStatus, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if !p.inited {
    return nil, fmt.Errorf("Status: Pool ist nicht initialisiert")
  }

  status := &ShmPoolStatus{
    Total:  MAX_POOLS,
    Blocks: make([]ShmBlockStatus, 0, MAX_POOLS),
  }

  for _, block := range p.blocks {
    if block == nil {
      continue
    }
    status.Blocks = append(status.Blocks, ShmBlockStatus{
      ID:       block.ID,
      Key:      block.Key,
      ShmID:    block.ShmID,
      InUse:    block.InUse,
      WorkerID: block.WorkerID,
    })
    if block.InUse {
      status.Used++
    }
  }

  status.Free = status.Total - status.Used
  return status, nil
}

//########################################
func (p *ShmPool) Cleanup() (int, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  return p.cleanupLocked()
}

//########################################
func (p *ShmPool) cleanupLocked() (int, error) {
  if !p.inited {
    return 0, nil
  }

  freed := 0
  var errs []error

  for i, block := range p.blocks {
    if block == nil {
      continue
    }
    if block.Data != nil {
      if err := ShmDt(block.Data); err != nil {
        errs = append(errs, fmt.Errorf("Cleanup: ShmDt für Block %d fehlgeschlagen: %v", i, err))
      }
    }
    if block.ShmID != 0 {
      if err := ShmRm(block.ShmID); err != nil {
        errs = append(errs, fmt.Errorf("Cleanup: ShmRm für Block %d fehlgeschlagen: %v", i, err))
      }
    }
    block.Data = nil
    block.InUse = false
    block.WorkerID = -1
    freed++
  }

  p.inited = false
  return freed, errors.Join(errs...)
}

//########################################
// ResetForTest bereinigt den globalen Pool und setzt ihn zurück.
// Nur für Tests!
func ResetForTest() error {
  poolMutex.Lock()
  defer poolMutex.Unlock()

  if globalPool != nil {
    globalPool.cleanupLocked()
  }
  globalPool = nil
  return nil
}
