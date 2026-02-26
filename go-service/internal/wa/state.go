package wa

import "sync/atomic"

type State struct {
	ready atomic.Bool
}

func NewState(initialReady bool) *State {
	s := &State{}
	s.ready.Store(initialReady)
	return s
}

func (s *State) Ready() bool {
	if s == nil {
		return false
	}
	return s.ready.Load()
}

func (s *State) SetReady(v bool) {
	if s == nil {
		return
	}
	s.ready.Store(v)
}
