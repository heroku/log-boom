package datastore

import "container/ring"

// MemoryDB implements an in memory Datastore.
type MemoryDB struct {
	keep  int
	rings map[string]*ring.Ring
}

// NewInMemory creates a new in memory Datastore.
func NewInMemory(keep int) (*MemoryDB, error) {
	db := &MemoryDB{
		keep:  keep,
		rings: make(map[string]*ring.Ring),
	}
	return db, nil
}

// Healthcheck always return true.
func (db *MemoryDB) Healthcheck() (bool, error) {
	return true, nil
}

// Insert inserts logs into in memory ring buffer.
func (db *MemoryDB) Insert(token string, lines []string) (int, error) {
	buf, ok := db.rings[token]
	if !ok {
		buf = ring.New(db.keep)
		db.rings[token] = buf
	}

	for _, line := range lines {
		buf.Value = line
		buf = buf.Next()
	}
	return len(lines), nil
}

// List lists the stored in memory logs
func (db *MemoryDB) List(token string) ([]string, error) {
	buf, ok := db.rings[token]
	if !ok {
		return nil, ErrNoSuchToken
	}

	lines := make([]string, 0, buf.Len())
	buf.Do(func(x interface{}) {
		if line, ok := x.(string); ok {
			lines = append(lines, line)
		}
	})
	return lines, nil
}
