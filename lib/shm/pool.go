//      lib/shm/pool.go
//  Autor    : Gerhard Quell - gquell@skequell.de
//  CoAutor  : claude 4.0 sonnet
//  Copyright: 2025 Gerhard Quell - SKEQuell
//  Erstellt : 20260711
//########################################

package shm

import (
  "fmt"
  "sync"
  // "shm"
  // "./shm"  // Deine eigene SHM-Implementation
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
  Data     []byte
  InUse    bool
  WorkerID int
}

//########################################
type ShmPool struct {
  blocks   [MAX_POOLS]*ShmBlock
  mutex    sync.Mutex
  inited   bool
}

//########################################
var globalPool *ShmPool
var once sync.Once

//########################################
func GetPool() (*ShmPool, error) {
  var initErr error
  once.Do(func() {
    globalPool = &ShmPool{}
    initErr = globalPool.Init()
  })
  if initErr != nil {
    return nil, initErr
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
  
  for i := 0; i < MAX_POOLS; i++ {
    key := SHM_BASE + i
    
    // SHM erstellen mit deiner API
    shmID, err := ShmGet(key, POOL_SIZE, IPC_CREAT|0644)
    if err != nil {
      return fmt.Errorf("ERR100: SHM create failed pool %d: %v", i, err)
    }
    
    // SHM anhängen
    data, err := ShmAt(shmID, 0, 0)
    if err != nil {
      return fmt.Errorf("ERR101: SHM attach failed pool %d: %v", i, err)
    }
    
    p.blocks[i] = &ShmBlock{
      ID:       i,
      Key:      key, 
      ShmID:    shmID,
      Data:     data,
      InUse:    false,
      WorkerID: -1,
    }
    
    fmt.Printf("SHM Pool %d: Key=0x%x, ShmID=%d, Size=%d bytes\n", 
               i, key, shmID, POOL_SIZE)
  }
  
  p.inited = true
  return nil
}

//########################################
func (p *ShmPool) Allocate(workerID int) (*ShmBlock, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  
  for i := 0; i < MAX_POOLS; i++ {
    if !p.blocks[i].InUse {
      p.blocks[i].InUse = true
      p.blocks[i].WorkerID = workerID
      return p.blocks[i], nil
    }
  }
  
  return nil, fmt.Errorf("ERR102: No free SHM blocks")
}

//########################################
func (p *ShmPool) Release(blockID int) error {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  
  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("ERR103: Invalid block ID %d", blockID)
  }
  
  if !p.blocks[blockID].InUse {
    return fmt.Errorf("ERR104: Block %d not in use", blockID)
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

  if blockID < 0 || blockID >= MAX_POOLS {
    return fmt.Errorf("ERR105: Invalid block ID %d", blockID)
  }
  
  block := p.blocks[blockID]
  if !block.InUse {
    return fmt.Errorf("ERR106: Block %d not allocated", blockID)
  }
  
  if len(data) > POOL_SIZE {
    return fmt.Errorf("ERR107: Data too large: %d > %d", len(data), POOL_SIZE)
  }
  
  copy(block.Data, data)
  return nil
}

//########################################
func (p *ShmPool) ReadData(blockID int, maxLen int) ([]byte, error) {
  p.mutex.Lock()
  defer p.mutex.Unlock()

  if blockID < 0 || blockID >= MAX_POOLS {
    return nil, fmt.Errorf("ERR108: Invalid block ID %d", blockID)
  }
  
  block := p.blocks[blockID]
  if !block.InUse {
    return nil, fmt.Errorf("ERR109: Block %d not allocated", blockID)
  }
  
  if maxLen > POOL_SIZE {
    maxLen = POOL_SIZE
  }
  
  result := make([]byte, maxLen)
  copy(result, block.Data[:maxLen])
  return result, nil
}

//########################################
func (p *ShmPool) Status() {
  p.mutex.Lock()
  defer p.mutex.Unlock()
  
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
  }
  p.inited = false
}
