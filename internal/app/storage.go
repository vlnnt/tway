package app

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type StateStorage struct {
	db *sql.DB
}

func NewStateStorage(
	path string,
) (*StateStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	storage := &StateStorage{
		db: db,
	}

	if err := storage.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return storage, nil
}

func (s *StateStorage) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS stream_states (
			channel TEXT PRIMARY KEY,
			is_live INTEGER NOT NULL,
			stream_id TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		);
	`)

	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	return nil
}

func (s *StateStorage) Get(
	channel string,
) (*StreamState, error) {
	row := s.db.QueryRow(`
		SELECT
			channel,
			is_live,
			stream_id,
			updated_at
		FROM stream_states
		WHERE channel = ?
	`, channel)

	var state StreamState
	var live int

	err := row.Scan(
		&state.Channel,
		&live,
		&state.StreamID,
		&state.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	state.IsLive = live == 1
	return &state, nil
}

func (s *StateStorage) Save(
	state StreamState,
) error {
	_, err := s.db.Exec(`
		INSERT INTO stream_states (
			channel,
			is_live,
			stream_id,
			updated_at
		)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(channel)
		DO UPDATE SET
			is_live = excluded.is_live,
			stream_id = excluded.stream_id,
			updated_at = excluded.updated_at
	`,
		state.Channel,
		boolToInt(state.IsLive),
		state.StreamID,
		state.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	return nil
}

func (s *StateStorage) Close() error {
	if s.db == nil {
		return nil
	}

	return s.db.Close()
}

func boolToInt(
	value bool,
) int {
	if value {
		return 1
	}

	return 0
}
