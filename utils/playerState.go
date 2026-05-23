package utils

import (
	"sync"
	"time"
)

type PropInventory map[int]int

type PlayerState struct {
	mu               sync.Mutex
	Inventory        PropInventory
	DoubleScoreUntil time.Time
	ShieldCount      int
}

func (ps *PlayerState) AddProp(propID int, count int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.Inventory == nil {
		ps.Inventory = make(PropInventory)
	}
	ps.Inventory[propID] += count
}

func (ps *PlayerState) UseProp(propID int) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.Inventory == nil {
		return false
	}
	if ps.Inventory[propID] > 0 {
		ps.Inventory[propID]--
		if ps.Inventory[propID] <= 0 {
			delete(ps.Inventory, propID)
		}
		return true
	}
	return false
}

func (ps *PlayerState) HasProp(propID int) bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.Inventory == nil {
		return false
	}
	return ps.Inventory[propID] > 0
}

func (ps *PlayerState) IsDoubleScoreActive() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return time.Now().Before(ps.DoubleScoreUntil)
}

func (ps *PlayerState) ActivateDoubleScore(duration time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	now := time.Now()
	if now.After(ps.DoubleScoreUntil) {
		ps.DoubleScoreUntil = now.Add(duration)
	} else {
		ps.DoubleScoreUntil = ps.DoubleScoreUntil.Add(duration)
	}
}

func (ps *PlayerState) GetDoubleScoreRemaining() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	remaining := time.Until(ps.DoubleScoreUntil)
	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds())
}

func (ps *PlayerState) AddShield(count int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.ShieldCount += count
}

func (ps *PlayerState) ConsumeShield() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.ShieldCount > 0 {
		ps.ShieldCount--
		return true
	}
	return false
}

func (ps *PlayerState) GetShieldCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.ShieldCount
}

func (ps *PlayerState) GetInventory() PropInventory {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	copy_ := make(PropInventory)
	for k, v := range ps.Inventory {
		copy_[k] = v
	}
	return copy_
}

func (ps *PlayerState) Clear() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Inventory = make(PropInventory)
	ps.DoubleScoreUntil = time.Time{}
	ps.ShieldCount = 0
}

// PlayerStateManager manages per-player state
type PlayerStateManager struct {
	mu    sync.RWMutex
	store map[int]*PlayerState
}

func NewPlayerStateManager() *PlayerStateManager {
	return &PlayerStateManager{
		store: make(map[int]*PlayerState),
	}
}

func (psm *PlayerStateManager) Get(playerID int) *PlayerState {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	ps, ok := psm.store[playerID]
	if !ok {
		ps = &PlayerState{Inventory: make(PropInventory)}
		psm.store[playerID] = ps
	}
	return ps
}

func (psm *PlayerStateManager) Delete(playerID int) {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	delete(psm.store, playerID)
}

func (psm *PlayerStateManager) ClearAll() {
	psm.mu.Lock()
	defer psm.mu.Unlock()
	psm.store = make(map[int]*PlayerState)
}
