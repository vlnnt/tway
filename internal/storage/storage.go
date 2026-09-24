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
			is_tracked INTEGER NOT NULL,
			is_live INTEGER NOT NULL,
			last_stream_at DATETIME NOT NULL,
			started_at DATETIME NOT NULL,

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
			is_tracked,
			is_live,
			last_stream_at,
			started_at
		FROM stream_states
		WHERE platform = ?
			AND channel = ?
	`, platform, channel)

	var streamState StreamState
	var isTracked int
	var isLive int

	err := row.Scan(
		&streamState.Platform,
		&streamState.Channel,
		&isTracked,
		&isLive,
		&streamState.LastStreamAt,
		&streamState.StartedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	streamState.IsLive = isLive == 1
	ss.logger.Info("Get state storage completed success",
		zap.Any("Stream state", streamState),
	)
	return &streamState, nil
}

func (ss *StateStorage) Ensure(
	platform, channel string,
	lastStreamAt time.Time,
) error {
	ss.logger.Info("Ensure state storage started",
		zap.String("Platform", platform),
		zap.String("Channel", channel),
		zap.Any("Last Stream Time", lastStreamAt),
	)
	ss.mu.Lock()
	defer ss.mu.Unlock()

	_, err := ss.db.Exec(`
		INSERT INTO stream_states (
			platform,
			channel,
			is_tracked,
			is_live,
			last_stream_at,
			started_at
		)
		VALUES (?, ?, 1, 0, ?, ?)
		ON CONFLICT(platform, channel)
		DO NOTHING
	`,
		platform,
		channel,
		lastStreamAt,
		time.Time{},
	)
	if err != nil {
		return fmt.Errorf("ensure state: %w", err)
	}

	ss.logger.Info("Ensure state storage completed success!")
	return nil
}

func (ss *StateStorage) SyncTracked(
	active []StreamKey,
) error {
	ss.logger.Info(
		"Sync tracked states started",
		zap.Int("Active", len(active)),
	)

	ss.mu.Lock()
	defer ss.mu.Unlock()

	tx, err := ss.db.Begin()
	if err != nil {
		return fmt.Errorf("begin sync tracked transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE stream_states
		SET is_tracked = 0
	`); err != nil {
		return fmt.Errorf("mark states untracked: %w", err)
	}

	for _, key := range active {
		_, err := tx.Exec(`
			INSERT INTO stream_states (
				platform,
				channel,
				is_tracked,
				is_live,
				last_stream_at,
				started_at
			)
			VALUES (?, ?, 1, 0, ?, ?)
			ON CONFLICT(platform, channel)
			DO UPDATE SET
				is_tracked = 1
		`,
			key.Platform,
			key.Channel,
			time.Time{},
			time.Time{},
		)
		if err != nil {
			return fmt.Errorf(
				"sync tracked state %s/%s: %w",
				key.Platform,
				key.Channel,
				err,
			)
		}
	}

	if _, err := tx.Exec(`
		UPDATE stream_states
		SET
			is_live = 0,
			started_at = ?
		WHERE is_tracked = 0
	`,
		time.Time{},
	); err != nil {
		return fmt.Errorf("reset untracked states: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sync tracked transaction: %w", err)
	}

	ss.logger.Info("Sync tracked states completed success!")
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
			last_stream_at = ?,
			started_at = ?
		WHERE platform = ?
			AND channel = ?`,
		boolToInt(state.IsLive),
		state.LastStreamAt,
		state.StartedAt,
		state.Platform,
		state.Channel,
	)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

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

func (ss *StateStorage) GetTracked() ([]StreamState, error) {
	ss.logger.Info("Get tracked state storage started!")

	ss.mu.RLock()
	defer ss.mu.RUnlock()

	rows, err := ss.db.Query(`
		SELECT
			platform,
			channel,
			is_tracked,
			is_live,
			last_stream_at,
			started_at
		FROM stream_states
		WHERE is_tracked = 1
	`)
	if err != nil {
		return nil, fmt.Errorf("get tracked states: %w", err)
	}
	defer rows.Close()

	var streamStates []StreamState
	for rows.Next() {
		var state StreamState
		var isTracked int
		var isLive int

		if err := rows.Scan(
			&state.Platform,
			&state.Channel,
			&isTracked,
			&isLive,
			&state.LastStreamAt,
			&state.StartedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tracked state: %w", err)
		}

		state.IsTracked = isTracked == 1
		state.IsLive = isLive == 1

		streamStates = append(
			streamStates,
			state,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tracked states: %w", err)
	}

	ss.logger.Info(
		"Get tracked state storage completed success!",
		zap.Int("States", len(streamStates)),
	)

	return streamStates, nil
}

func (ss *StateStorage) GetAll() ([]StreamState, error) {
	ss.logger.Info("Get all state storage started!")
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	rows, err := ss.db.Query(`
		SELECT
			platform,
			channel,
			is_tracked,
			is_live,
			last_stream_at,
			started_at
		FROM stream_states
	`)
	if err != nil {
		return nil, fmt.Errorf("get all states: %w", err)
	}
	defer rows.Close()

	var streamState []StreamState
	for rows.Next() {
		var state StreamState
		var isTracked int
		var isLive int

		if err := rows.Scan(
			&state.Platform,
			&state.Channel,
			&isTracked,
			&isLive,
			&state.LastStreamAt,
			&state.StartedAt,
		); err != nil {
			return nil, fmt.Errorf("scan state: %w", err)
		}

		state.IsTracked = isTracked == 1
		state.IsLive = isLive == 1
		streamState = append(streamState, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate states: %w", err)
	}

	ss.logger.Info(
		"Get all state storage completed success!",
		zap.Int("States", len(streamState)),
	)
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
