//      lib/shm/pool.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude sonnet 4.6
//  Copyright: 2026 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//########################################

package shm

import (
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
type ShmPool struct {
  blocks   [MAX_POOLS]*ShmBlock
  mutex    sync.Mutex
  inited   bool
  cleanedUp bool
}

//########################################
var globalPool *ShmPool
var once sync.Once
var initErr error

//########################################
func GetPool() (*ShmPool, error) {
  once.Do(func() {
    p := &ShmPool{}
    if err := p.Init(); err != nil {
      initErr = err
      return
    }
    globalPool = p
  })
  if initErr != nil {
    return nil, initErr
  }
  if globalPool != nil && globalPool.cleanedUp {
    return nil, fmt.Errorf("shared memory pool has been cleaned up")
  }
  return globalPool, nil
}

//########################################
func (p *ShmPool) Init() error {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  
  if p.inited {
    return nil
  }
  
  created := make([]*ShmBlock, 0, MAX_POOLS)

  for i := 0; i < MAX_POOLS; i++ {
    key := SHM_BASE + i

    // SHM erstellen mit deiner API
    shmID, err := ShmGet(key, POOL_SIZE, IPC_CREAT|0644)
    if err != nil {
      p.rollback(created)
      return fmt.Errorf("ERR100: SHM create failed pool %d: %v", i, err)
    }

    // SHM anhängen
    data, err := ShmAt(shmID, 0, 0, POOL_SIZE)
    if err != nil {
      ShmRm(shmID)
      p.rollback(created)
      return fmt.Errorf("ERR101: SHM attach failed pool %d: %v", i, err)
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

    // Block initialisiert, kein stdout-Output im Initialisierungsloop.
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

  if p.cleanedUp {
    return nil, fmt.Errorf("shared memory pool has been cleaned up")
  }

  if !p.inited {
    return nil, fmt.Errorf("shared memory pool not initialized")
  }

  for i := 0; i < MAX_POOLS; i++ {
    if !p.blocks[i].InUse {
      p.blocks[i].InUse = true
      p.blocks[i].WorkerID = workerID
      return p.blocks[i], nil
    }
  }

  return nil, fmt.Errorf("no free shared-memory blocks")
}

//########################################
func (p *ShmPool) Release(blockID int) error {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if p.cleanedUp {
    return fmt.Errorf("shared memory pool has been cleaned up")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("invalid block id %d", blockID)
  }

  if !p.blocks[blockID].InUse {
    return fmt.Errorf("block %d not in use", blockID)
  }

  p.blocks[blockID].InUse = false
  p.blocks[blockID].WorkerID = -1

  // SHM nullen
  for i := range p.blocks[blockID].Data {
    p.blocks[blockID].Data[i] = 0
  }

  return nil
}

//########################################
func (p *ShmPool) WriteData(blockID int, data []byte) error {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if p.cleanedUp {
    return fmt.Errorf("shared memory pool has been cleaned up")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("invalid block id %d", blockID)
  }

  block := p.blocks[blockID]
  if !block.InUse {
    return fmt.Errorf("block %d not allocated", blockID)
  }

  if len(data) > block.Size {
    return fmt.Errorf("data too large: %d > %d", len(data), block.Size)
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

  if p.cleanedUp {
    return nil, fmt.Errorf("shared memory pool has been cleaned up")
  }

  if blockID < 0 || blockID >= MAX_POOLS {
    return nil, fmt.Errorf("invalid block id %d", blockID)
  }

  block := p.blocks[blockID]
  if !block.InUse {
    return nil, fmt.Errorf("block %d not allocated", blockID)
  }

  if maxLen > block.Size {
    maxLen = block.Size
  }

  result := make([]byte, maxLen)
  copy(result, block.Data[:maxLen])
  return result, nil
}

//########################################
func (p *ShmPool) Status() {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if p.cleanedUp {
    fmt.Println("=== SHM POOL STATUS (cleaned up) ===")
    return
  }

  fmt.Println("=== SHM POOL STATUS ===")
  for i, block := range p.blocks {
    status := "FREE"
    worker := "-"
    if block.InUse {
      status = "USED"
      worker = fmt.Sprintf("%d", block.WorkerID)
    }
    fmt.Printf("Pool %d: Key=0x%x, ShmID=%d, Status=%s, Worker=%s\n",
               i, block.Key, block.ShmID, status, worker)
  }
}

//########################################
func (p *ShmPool) Cleanup() {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  for i, block := range p.blocks {
    if block != nil && block.Data != nil {
      ShmDt(block.Data)
      ShmRm(block.ShmID)
      fmt.Printf("SHM Pool %d cleaned up\n", i)
    }
    if block != nil {
      block.Data = nil
      block.InUse = false
      block.WorkerID = -1
    }
  }
  p.inited = false
  p.cleanedUp = true
}
