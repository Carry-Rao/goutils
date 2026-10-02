package mixture

import (
	"errors"
	"testing"

	"github.com/Carry-Rao/goutils/database/cache"
)

type rec struct {
	ID   int `db:",primary"`
	Name string
}

// stub is a cache.Table whose behaviour each test dictates.
type stub struct {
	data   map[int]rec
	getErr error
	setErr error
	insErr error
	delErr error
	reads  int
}

func newStub() *stub { return &stub{data: map[int]rec{}} }

func (s *stub) Ins(v rec) error {
	if s.insErr != nil {
		return s.insErr
	}
	s.data[v.ID] = v
	return nil
}

func (s *stub) Get(v rec) ([]rec, error) {
	s.reads++
	if s.getErr != nil {
		return nil, s.getErr
	}
	if got, ok := s.data[v.ID]; ok {
		return []rec{got}, nil
	}
	return nil, nil
}

func (s *stub) Set(old, updated rec) error {
	if s.setErr != nil {
		return s.setErr
	}
	s.data[updated.ID] = updated
	return nil
}

func (s *stub) Del(v rec) error {
	if s.delErr != nil {
		return s.delErr
	}
	delete(s.data, v.ID)
	return nil
}

func TestChainFallsThrough(t *testing.T) {
	miss, hit := newStub(), newStub()
	hit.data[7] = rec{ID: 7, Name: "found"}

	m := New[rec]()
	m.Add(miss, Continue)
	m.Add(hit, Continue)

	got, err := m.Table().Get(rec{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "found" {
		t.Fatalf("got %+v", got)
	}
	if miss.reads != 1 {
		t.Errorf("first layer should be consulted once, got %d", miss.reads)
	}
}

// A layer marked Return aborts the chain on failure.
func TestChainReturnStops(t *testing.T) {
	boom := errors.New("boom")
	first, second := newStub(), newStub()
	first.getErr = boom

	m := New[rec]()
	m.Add(first, Return)
	m.Add(second, Continue)

	if _, err := m.Table().Get(rec{ID: 1}); !errors.Is(err, boom) {
		t.Errorf("want boom, got %v", err)
	}
	if second.reads != 0 {
		t.Error("later layers must not be consulted after Return")
	}
}

// Continue keeps going, and the last error surfaces if nothing matches.
func TestChainContinuesOnError(t *testing.T) {
	boom := errors.New("down")
	first, second := newStub(), newStub()
	first.getErr = boom

	m := New[rec]()
	m.Add(first, Continue)
	m.Add(second, Continue)

	got, err := m.Table().Get(rec{ID: 1})
	if !errors.Is(err, boom) {
		t.Errorf("want boom, got rows %+v err %v", got, err)
	}
	if second.reads != 1 {
		t.Error("second layer should have been tried")
	}
}

// Writes reach every layer, so a filter, a cache and a store all stay in sync.
func TestChainWriteReachesEveryLayer(t *testing.T) {
	first, second := newStub(), newStub()

	m := New[rec]()
	m.Add(first, Continue)
	m.Add(second, Continue)

	if err := m.Table().Ins(rec{ID: 1, Name: "a"}); err != nil {
		t.Fatal(err)
	}
	if len(first.data) != 1 {
		t.Error("first layer should hold the row")
	}
	if len(second.data) != 1 {
		t.Error("later layers must also receive the write")
	}
}

func TestEmptyChain(t *testing.T) {
	m := New[rec]()
	if _, err := m.Table().Get(rec{ID: 1}); err != nil {
		t.Errorf("empty chain Get returned %v", err)
	}
	if err := m.Table().Ins(rec{ID: 1}); err != nil {
		t.Errorf("empty chain Ins returned %v", err)
	}
}

// Close reports every layer failure rather than only the first.
func TestCloseReportsEveryLayer(t *testing.T) {
	b1 := errors.New("b1")
	b2 := errors.New("b2")

	m := New[rec]()
	m.Add(closable{err: b1}, Continue)
	m.Add(closable{err: b2}, Continue)

	err := m.Close()
	if !errors.Is(err, b1) || !errors.Is(err, b2) {
		t.Errorf("want both errors, got %v", err)
	}
}

type closable struct{ err error }

func (c closable) Ins(rec) error          { return nil }
func (c closable) Get(rec) ([]rec, error) { return nil, nil }
func (c closable) Set(rec, rec) error     { return nil }
func (c closable) Del(rec) error          { return nil }
func (c closable) Close() error           { return c.err }

var _ cache.Table[rec] = (*stub)(nil)
