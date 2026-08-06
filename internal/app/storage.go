package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type StateStorage struct {
	path string
}

func NewStateStorage(
	path string,
) *StateStorage {
	return &StateStorage{
		path: path,
	}
}

func (s *StateStorage) Load() (States, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(States), nil
		}

		return nil, fmt.Errorf("open state file: %w", err)
	}
	defer file.Close()

	var states States
	err = json.NewDecoder(file).Decode(&states)
	if err != nil {
		return nil, fmt.Errorf("decode state: %w")
	}

	return states, nil
}

func (s *StateStorage) Save(
	states States,
) error {
	dir := filepath.Dir(s.path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	file, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("create state file: %w")
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(states)
	if err != nil {
		return fmt.Errorf("encode state: %w")
	}

	return nil
}
