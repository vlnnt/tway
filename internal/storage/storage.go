package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

type StateStorage struct {
	logger *zap.Logger
	db     *sql.DB
	mu     sync.RWMutex
}

func NewStateStorage(
	logger *zap.Logger,
	path string,
) (*StateStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	storage := &StateStorage{
		logger: logger,
		db:     db,
	}

	if err := storage.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return storage, nil
}

func (ss *StateStorage) migrate() error {
	ss.logger.Info("Migrate database started!")
	_, err := ss.db.Exec(`
		CREATE TABLE IF NOT EXISTS stream_states (
			platform TEXT NOT NULL,
			channel TEXT NOT NULL,
			is_live INTEGER NOT NULL,
			stream_id TEXT NOT NULL,
			updated_at DATETIME NOT NULL,

			PRIMARY KEY (platform, channel)
		);
	`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	if _, err := ss.db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := ss.db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return fmt.Errorf("set busy timeout: %w", err)
	}

	ss.logger.Info("Migrate database completed success!")
	return nil
}

func (ss *StateStorage) Get(
	platform, channel string,
) (*StreamState, error) {
	ss.logger.Info("Get state storage started",
		zap.String("Platform", platform),
		zap.String("Channel", channel),
	)
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	row := ss.db.QueryRow(`
		SELECT
			platform,
			channel,
			is_live,
			stream_id,
			updated_at
		FROM stream_states
		WHERE platform = ?
			AND channel = ?
	`, platform, channel)

	var streamState StreamState
	var live int

	err := row.Scan(
		&streamState.Platform,
		&streamState.Channel,
		&live,
		&streamState.StreamID,
		&streamState.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	streamState.IsLive = live == 1
	ss.logger.Info("Get state storage completed success",
		zap.Any("Stream state", streamState),
	)
	return &streamState, nil
}

func (ss *StateStorage) Ensure(
	platform, channel string,
) error {
	ss.logger.Info("Ensure state storage started",
		zap.String("Platform", platform),
		zap.String("Channel", channel),
	)
	ss.mu.Lock()
	defer ss.mu.Unlock()

	_, err := ss.db.Exec(`
		INSERT INTO stream_states (
			platform,
			channel,
			is_live,
			stream_id,
			updated_at
		)
		VALUES (?, ?, 0, '', ?)
		ON CONFLICT(platform, channel)
		DO NOTHING
	`,
		platform,
		channel,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("ensure state: %w", err)
	}

	ss.logger.Info("Ensure state storage completed success!")
	return nil
}

func (ss *StateStorage) Update(
	state StreamState,
) error {
	ss.logger.Info("Update state storage started",
		zap.Any("State", state))
	ss.mu.Lock()
	defer ss.mu.Unlock()

	result, err := ss.db.Exec(`
		UPDATE stream_states
		SET
			is_live = ?,
			stream_id = ?,
			updated_at = ?
		WHERE platform = ?
			AND channel = ?`,
		boolToInt(state.IsLive),
		state.StreamID,
		state.UpdatedAt,
		state.Platform,
		state.Channel,
	)

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf(
			"stream state not found: %s/%s",
			state.Platform,
			state.Channel,
		)
	}

	ss.logger.Info("Update state storage completed success!")
	return nil
}

func (ss *StateStorage) GetAll() ([]StreamState, error) {
	ss.logger.Info("Get all state storage started!")
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	rows, err := ss.db.Query(`
		SELECT
			platform,
			channel,
			is_live,
			stream_id,
			updated_at
		FROM stream_states
	`)
	if err != nil {
		return nil, fmt.Errorf("get all states: %w", err)
	}
	defer rows.Close()

	var streamState []StreamState
	for rows.Next() {

		var state StreamState
		var live int

		if err := rows.Scan(
			&state.Platform,
			&state.Channel,
			&live,
			&state.StreamID,
			&state.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan state: %w", err)
		}

		state.IsLive = live == 1
		streamState = append(streamState, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate states: %w", err)
	}

	ss.logger.Info("Get all state storage completed success!")
	return streamState, nil
}

func (ss *StateStorage) Close() error {
	if ss.db == nil {
		return nil
	}

	return ss.db.Close()
}

func boolToInt(
	value bool,
) int {
	if value {
		return 1
	}

	return 0
}
