package portal

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
)

type MapStore struct {
	path     string
	lockPath string
}

// NewMapStore returns a store for the RFID-to-music map at path. All writes take
// an exclusive flock on lockPath, which Python's rfid_map_store uses too.
func NewMapStore(path, lockPath string) *MapStore {
	return &MapStore{path: path, lockPath: lockPath}
}

// Load reads the map. A missing or empty file is an empty map.
func (m *MapStore) Load() (map[string]string, error) {
	raw, err := os.ReadFile(m.path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read map: %w", err)
	}
	out := map[string]string{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse map: %w", err)
	}
	return out, nil
}

// Update runs mutate inside an exclusive lock and persists the result atomically.
func (m *MapStore) Update(mutate func(map[string]string) error) error {
	lock, err := os.OpenFile(m.lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock: %w", err)
	}
	defer func() { _ = lock.Close() }()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN) }()

	data, err := m.Load()
	if err != nil {
		return err
	}
	if err := mutate(data); err != nil {
		return err
	}
	return m.writeAtomic(data)
}

// Save replaces the whole map with data.
func (m *MapStore) Save(data map[string]string) error {
	return m.Update(func(current map[string]string) error {
		for k := range current {
			delete(current, k)
		}
		for k, v := range data {
			current[k] = v
		}
		return nil
	})
}

// Assign points cardID at folder.
func (m *MapStore) Assign(cardID, folder string) error {
	return m.Update(func(data map[string]string) error {
		data[cardID] = folder
		return nil
	})
}

// Remove deletes cardID from the map.
func (m *MapStore) Remove(cardID string) error {
	return m.Update(func(data map[string]string) error {
		delete(data, cardID)
		return nil
	})
}

func (m *MapStore) writeAtomic(data map[string]string) error {
	buf, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return writeFileAtomic(m.path, string(buf))
}
