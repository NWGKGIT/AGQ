package store

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"agq-daemon/internal/domain"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	email      TEXT UNIQUE NOT NULL,
	plan_name  TEXT,
	first_seen DATETIME NOT NULL,
	last_seen  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS snapshots (
	id                       INTEGER PRIMARY KEY AUTOINCREMENT,
	account_id               INTEGER NOT NULL REFERENCES accounts(id),
	captured_at              DATETIME NOT NULL,
	prompt_credits_available INTEGER,
	prompt_credits_monthly   INTEGER,
	flow_credits_available   INTEGER,
	flow_credits_monthly     INTEGER,
	raw_json                 TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS model_quotas (
	id                  INTEGER PRIMARY KEY AUTOINCREMENT,
	snapshot_id         INTEGER NOT NULL REFERENCES snapshots(id),
	label               TEXT NOT NULL,
	model_id            TEXT NOT NULL,
	remaining_fraction  REAL,
	remaining_pct       REAL,
	is_exhausted        BOOLEAN,
	reset_time          DATETIME,
	pool_reset_time     DATETIME,
	time_until_reset_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_snapshots_account_captured
	ON snapshots(account_id, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_model_quotas_snapshot
	ON model_quotas(snapshot_id);
`

var sqliteIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// DB wraps *sql.DB with domain-aware query methods.
type DB struct {
	sql *sql.DB
}

// Open opens or creates the SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	opened := false
	defer func() {
		if !opened {
			_ = sqlDB.Close()
		}
	}()

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, pragma := range pragmas {
		if _, err := sqlDB.Exec(pragma); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", pragma, err)
		}
	}

	if _, err := sqlDB.Exec(schema); err != nil {
		return nil, fmt.Errorf("schema migration: %w", err)
	}

	db := &DB{sql: sqlDB}
	if err := db.addColumnIfMissing("model_quotas", "pool_reset_time", "DATETIME"); err != nil {
		return nil, fmt.Errorf("column migration: %w", err)
	}

	opened = true
	return db, nil
}

// Close releases the underlying database connection.
func (d *DB) Close() error {
	return d.sql.Close()
}

// SaveSnapshot persists one poll result in a single transaction.
func (d *DB) SaveSnapshot(s domain.QuotaSnapshot) error {
	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	ts := s.CapturedAt.UTC().Format(time.RFC3339)

	_, err = tx.Exec(`
		INSERT INTO accounts (email, plan_name, first_seen, last_seen)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email) DO UPDATE SET
			plan_name = excluded.plan_name,
			last_seen = excluded.last_seen
	`, s.Email, s.PlanName, ts, ts)
	if err != nil {
		return fmt.Errorf("upsert account: %w", err)
	}

	var accountID int64
	if err = tx.QueryRow(`SELECT id FROM accounts WHERE email = ?`, s.Email).Scan(&accountID); err != nil {
		return fmt.Errorf("get account id: %w", err)
	}

	res, err := tx.Exec(`
		INSERT INTO snapshots
			(account_id, captured_at, prompt_credits_available, prompt_credits_monthly,
			 flow_credits_available, flow_credits_monthly, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, accountID, ts,
		s.PromptCreditsAvailable, s.PromptCreditsMonthly,
		s.FlowCreditsAvailable, s.FlowCreditsMonthly,
		s.RawJSON)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}

	snapshotID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("last insert id: %w", err)
	}

	for _, m := range s.Models {
		var resetStr *string
		if m.ResetTime != nil {
			v := m.ResetTime.UTC().Format(time.RFC3339)
			resetStr = &v
		}

		var poolResetStr *string
		if m.PoolResetTime != nil {
			v := m.PoolResetTime.UTC().Format(time.RFC3339)
			poolResetStr = &v
		}

		_, err = tx.Exec(`
			INSERT INTO model_quotas
				(snapshot_id, label, model_id, remaining_fraction, remaining_pct,
				 is_exhausted, reset_time, pool_reset_time, time_until_reset_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, snapshotID, m.Label, m.ModelID,
			m.RemainingFraction, m.RemainingPct,
			m.IsExhausted, resetStr, poolResetStr, m.TimeUntilResetMs)
		if err != nil {
			return fmt.Errorf("insert model quota %q: %w", m.Label, err)
		}
	}

	return tx.Commit()
}

// GetAllAccounts returns every known account ordered by most-recently-seen.
func (d *DB) GetAllAccounts() ([]domain.AccountSummary, error) {
	rows, err := d.sql.Query(`
		SELECT id, email, COALESCE(plan_name, ''), first_seen, last_seen
		FROM accounts
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []domain.AccountSummary
	for rows.Next() {
		var a domain.AccountSummary
		var firstSeen, lastSeen string
		if err := rows.Scan(&a.ID, &a.Email, &a.PlanName, &firstSeen, &lastSeen); err != nil {
			return nil, err
		}
		if a.FirstSeen, err = parseDBTime(firstSeen, "first_seen"); err != nil {
			return nil, err
		}
		if a.LastSeen, err = parseDBTime(lastSeen, "last_seen"); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// GetAccount returns the account matching email, or nil, nil when no such
// account is known to the store.
func (d *DB) GetAccount(email string) (*domain.AccountSummary, error) {
	var (
		a                   domain.AccountSummary
		firstSeen, lastSeen string
	)
	err := d.sql.QueryRow(`
		SELECT id, email, COALESCE(plan_name, ''), first_seen, last_seen
		FROM accounts
		WHERE email = ?
	`, email).Scan(&a.ID, &a.Email, &a.PlanName, &firstSeen, &lastSeen)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if a.FirstSeen, err = parseDBTime(firstSeen, "first_seen"); err != nil {
		return nil, err
	}
	if a.LastSeen, err = parseDBTime(lastSeen, "last_seen"); err != nil {
		return nil, err
	}
	return &a, nil
}

// GetLatestSnapshot returns the most recent snapshot for email, including its
// model quota rows. It returns nil, nil when the account has no snapshots.
func (d *DB) GetLatestSnapshot(email string) (*domain.QuotaSnapshot, error) {
	var (
		id            int64
		capturedAt    string
		planName      string
		promptAvail   int64
		promptMonthly int64
		flowAvail     int64
		flowMonthly   int64
		rawJSON       string
	)
	err := d.sql.QueryRow(`
		SELECT s.id, s.captured_at,
		       s.prompt_credits_available, s.prompt_credits_monthly,
		       s.flow_credits_available,   s.flow_credits_monthly,
		       s.raw_json, COALESCE(a.plan_name, '')
		FROM snapshots s
		JOIN accounts a ON a.id = s.account_id
		WHERE a.email = ?
		ORDER BY s.captured_at DESC
		LIMIT 1
	`, email).Scan(&id, &capturedAt, &promptAvail, &promptMonthly,
		&flowAvail, &flowMonthly, &rawJSON, &planName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	s := &domain.QuotaSnapshot{
		Email:                  email,
		PlanName:               planName,
		PromptCreditsAvailable: promptAvail,
		PromptCreditsMonthly:   promptMonthly,
		FlowCreditsAvailable:   flowAvail,
		FlowCreditsMonthly:     flowMonthly,
		RawJSON:                rawJSON,
	}
	if s.CapturedAt, err = parseDBTime(capturedAt, "captured_at"); err != nil {
		return nil, err
	}
	if s.Models, err = d.getModelQuotas(id); err != nil {
		return nil, err
	}

	return s, nil
}

// GetSnapshotHistory returns paginated snapshot history for email, ordered
// newest-first. Pass before=zero to start from the current time.
func (d *DB) GetSnapshotHistory(email string, limit int, before time.Time) ([]domain.QuotaSnapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if before.IsZero() {
		before = time.Now().Add(time.Second)
	}

	rows, err := d.sql.Query(`
		SELECT s.id, s.captured_at,
		       s.prompt_credits_available, s.prompt_credits_monthly,
		       s.flow_credits_available,   s.flow_credits_monthly,
		       s.raw_json, COALESCE(a.plan_name, '')
		FROM snapshots s
		JOIN accounts a ON a.id = s.account_id
		WHERE a.email = ? AND s.captured_at < ?
		ORDER BY s.captured_at DESC
		LIMIT ?
	`, email, before.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []domain.QuotaSnapshot
	for rows.Next() {
		var (
			id            int64
			capturedAt    string
			promptAvail   int64
			promptMonthly int64
			flowAvail     int64
			flowMonthly   int64
			rawJSON       string
			planName      string
		)
		if err := rows.Scan(&id, &capturedAt, &promptAvail, &promptMonthly,
			&flowAvail, &flowMonthly, &rawJSON, &planName); err != nil {
			return nil, err
		}
		s := domain.QuotaSnapshot{
			Email:                  email,
			PlanName:               planName,
			PromptCreditsAvailable: promptAvail,
			PromptCreditsMonthly:   promptMonthly,
			FlowCreditsAvailable:   flowAvail,
			FlowCreditsMonthly:     flowMonthly,
			RawJSON:                rawJSON,
		}
		if s.CapturedAt, err = parseDBTime(capturedAt, "captured_at"); err != nil {
			return nil, err
		}
		if s.Models, err = d.getModelQuotas(id); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}
	return snapshots, rows.Err()
}

// GetLatestModelQuotas returns each model's quota data from the most recent
// snapshot of each account.
func (d *DB) GetLatestModelQuotas() ([]domain.ModelQuotaAggregate, error) {
	rows, err := d.sql.Query(`
		SELECT mq.label, mq.model_id,
		       mq.remaining_fraction, mq.remaining_pct,
		       mq.is_exhausted, mq.reset_time, mq.pool_reset_time,
		       a.email, s.captured_at
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		JOIN accounts a  ON a.id = s.account_id
		INNER JOIN (
			SELECT account_id, MAX(captured_at) AS max_cap
			FROM snapshots
			GROUP BY account_id
		) latest ON latest.account_id = s.account_id
		         AND latest.max_cap    = s.captured_at
		ORDER BY a.email, mq.label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	var results []domain.ModelQuotaAggregate
	for rows.Next() {
		var m domain.ModelQuotaAggregate
		var capturedAtStr string
		if err := rows.Scan(&m.Label, &m.ModelID,
			&m.RemainingFraction, &m.RemainingPct,
			&m.IsExhausted, &m.ResetTime, &m.PoolResetTime,
			&m.Email, &capturedAtStr); err != nil {
			return nil, err
		}
		m.CapturedAt = capturedAtStr
		if t, err := parseDBTime(capturedAtStr, "captured_at"); err == nil {
			m.StalenessSeconds = now.Unix() - t.Unix()
		} else {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// GetFractionSamplesSince returns every non-null model remaining fraction
// captured at or after since, across all accounts, ordered oldest-first. It
// backs the analytics time-series endpoint, which aggregates by day and
// provider.
func (d *DB) GetFractionSamplesSince(since time.Time) ([]domain.FractionSample, error) {
	rows, err := d.sql.Query(`
		SELECT s.captured_at, mq.label, mq.model_id, mq.remaining_fraction
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		WHERE s.captured_at >= ? AND mq.remaining_fraction IS NOT NULL
		ORDER BY s.captured_at ASC
	`, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFractionSamples(rows)
}

// GetAccountFractionSamplesSince returns one email's model remaining fractions
// captured at or after since, ordered by model then time. Null fractions are
// included so the sparkline endpoint can render gaps.
func (d *DB) GetAccountFractionSamplesSince(email string, since time.Time) ([]domain.FractionSample, error) {
	rows, err := d.sql.Query(`
		SELECT s.captured_at, mq.label, mq.model_id, mq.remaining_fraction
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		JOIN accounts a  ON a.id = s.account_id
		WHERE a.email = ? AND s.captured_at >= ?
		ORDER BY mq.model_id ASC, s.captured_at ASC
	`, email, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFractionSamples(rows)
}

// GetSnapshotRefsSince returns lightweight (id, captured_at) references for one
// email's snapshots captured at or after since, ordered oldest-first. The
// timeline endpoint uses these to infer session boundaries before loading model
// quotas only for the boundary snapshots.
func (d *DB) GetSnapshotRefsSince(email string, since time.Time) ([]domain.SnapshotRef, error) {
	rows, err := d.sql.Query(`
		SELECT s.id, s.captured_at
		FROM snapshots s
		JOIN accounts a ON a.id = s.account_id
		WHERE a.email = ? AND s.captured_at >= ?
		ORDER BY s.captured_at ASC
	`, email, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []domain.SnapshotRef
	for rows.Next() {
		var (
			ref        domain.SnapshotRef
			capturedAt string
		)
		if err := rows.Scan(&ref.ID, &capturedAt); err != nil {
			return nil, err
		}
		if ref.CapturedAt, err = parseDBTime(capturedAt, "captured_at"); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

// GetSnapshotModels returns the model quota rows for one snapshot id.
func (d *DB) GetSnapshotModels(snapshotID int64) ([]domain.ModelQuota, error) {
	return d.getModelQuotas(snapshotID)
}

// GetBreakdown returns one row per account/model pair, pairing each model's
// current remaining fraction (from the newest snapshot) with its starting
// fraction (the earliest snapshot sharing the current reset time). Consumed is
// the difference. Rows lacking a reset time or a distinct earlier snapshot in
// the current cycle leave StartingFraction and Consumed nil.
func (d *DB) GetBreakdown() ([]domain.BreakdownRow, error) {
	starts, err := d.cycleStartFractions()
	if err != nil {
		return nil, err
	}

	rows, err := d.sql.Query(`
		SELECT a.email, mq.label, mq.model_id,
		       mq.remaining_fraction, mq.reset_time, s.captured_at
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		JOIN accounts a  ON a.id = s.account_id
		INNER JOIN (
			SELECT account_id, MAX(captured_at) AS max_cap
			FROM snapshots
			GROUP BY account_id
		) latest ON latest.account_id = s.account_id
		         AND latest.max_cap    = s.captured_at
		ORDER BY a.email, mq.label
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.BreakdownRow
	for rows.Next() {
		var (
			row        domain.BreakdownRow
			resetStr   *string
			capturedAt string
		)
		if err := rows.Scan(&row.Email, &row.Label, &row.ModelID,
			&row.CurrentFraction, &resetStr, &capturedAt); err != nil {
			return nil, err
		}
		if resetStr != nil {
			t, err := parseDBTime(*resetStr, "reset_time")
			if err != nil {
				return nil, err
			}
			row.ResetTime = &t

			// Starting fraction comes from the earliest snapshot in the current
			// cycle (same reset time). It only counts when that snapshot is
			// strictly older than the current one, so a single-snapshot cycle
			// reports no consumption rather than a guessed zero.
			if start, ok := starts[cycleKey(row.Email, row.ModelID, *resetStr)]; ok &&
				start.capturedAt < capturedAt &&
				start.fraction != nil && row.CurrentFraction != nil {
				startFrac := *start.fraction
				consumed := startFrac - *row.CurrentFraction
				row.StartingFraction = &startFrac
				row.Consumed = &consumed
			}
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

type cycleStart struct {
	fraction   *float64
	capturedAt string
}

// cycleStartFractions returns, per account/model/reset-time, the remaining
// fraction and capture time of the earliest snapshot in that cycle.
func (d *DB) cycleStartFractions() (map[string]cycleStart, error) {
	rows, err := d.sql.Query(`
		SELECT a.email, mq.model_id, mq.reset_time,
		       mq.remaining_fraction, s.captured_at
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		JOIN accounts a  ON a.id = s.account_id
		INNER JOIN (
			SELECT s2.account_id, mq2.model_id, mq2.reset_time,
			       MIN(s2.captured_at) AS min_cap
			FROM model_quotas mq2
			JOIN snapshots s2 ON s2.id = mq2.snapshot_id
			WHERE mq2.reset_time IS NOT NULL
			GROUP BY s2.account_id, mq2.model_id, mq2.reset_time
		) firstc ON firstc.account_id = s.account_id
		        AND firstc.model_id   = mq.model_id
		        AND firstc.reset_time = mq.reset_time
		        AND firstc.min_cap    = s.captured_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	starts := make(map[string]cycleStart)
	for rows.Next() {
		var (
			email, modelID, resetStr, capturedAt string
			fraction                             *float64
		)
		if err := rows.Scan(&email, &modelID, &resetStr, &fraction, &capturedAt); err != nil {
			return nil, err
		}
		starts[cycleKey(email, modelID, resetStr)] = cycleStart{
			fraction:   fraction,
			capturedAt: capturedAt,
		}
	}
	return starts, rows.Err()
}

// GetAnalyticsStats computes the four headline analytics figures relative to
// now: polls in the last seven days, the most depleted model, the account with
// the most remaining quota, and the soonest upcoming reset.
func (d *DB) GetAnalyticsStats(now time.Time) (domain.AnalyticsStats, error) {
	var stats domain.AnalyticsStats

	weekAgo := now.UTC().AddDate(0, 0, -7).Format(time.RFC3339)
	if err := d.sql.QueryRow(
		`SELECT COUNT(*) FROM snapshots WHERE captured_at >= ?`, weekAgo,
	).Scan(&stats.TotalPollsThisWeek); err != nil {
		return domain.AnalyticsStats{}, err
	}

	rows, err := d.sql.Query(`
		SELECT a.email, mq.label, mq.remaining_fraction, mq.reset_time
		FROM model_quotas mq
		JOIN snapshots s ON s.id = mq.snapshot_id
		JOIN accounts a  ON a.id = s.account_id
		INNER JOIN (
			SELECT account_id, MAX(captured_at) AS max_cap
			FROM snapshots
			GROUP BY account_id
		) latest ON latest.account_id = s.account_id
		         AND latest.max_cap    = s.captured_at
	`)
	if err != nil {
		return domain.AnalyticsStats{}, err
	}
	defer rows.Close()

	type accountAgg struct {
		sum   float64
		count int
	}
	perAccount := make(map[string]*accountAgg)

	var nextReset time.Time // zero until the first upcoming reset is seen

	for rows.Next() {
		var (
			email, label string
			fraction     *float64
			resetStr     *string
		)
		if err := rows.Scan(&email, &label, &fraction, &resetStr); err != nil {
			return domain.AnalyticsStats{}, err
		}

		if fraction != nil {
			if stats.MostDepletedModel == nil || *fraction < stats.MostDepletedModel.RemainingFraction {
				stats.MostDepletedModel = &domain.DepletedModel{
					Email:             email,
					Label:             label,
					RemainingFraction: *fraction,
				}
			}
			agg := perAccount[email]
			if agg == nil {
				agg = &accountAgg{}
				perAccount[email] = agg
			}
			agg.sum += *fraction
			agg.count++
		}

		if resetStr != nil {
			reset, err := parseDBTime(*resetStr, "reset_time")
			if err != nil {
				return domain.AnalyticsStats{}, err
			}
			if !reset.Before(now) && (nextReset.IsZero() || reset.Before(nextReset)) {
				nextReset = reset
				stats.NextReset = &domain.NextReset{
					Email:     email,
					Label:     label,
					ResetTime: reset.UTC().Format(time.RFC3339),
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return domain.AnalyticsStats{}, err
	}

	for email, agg := range perAccount {
		avg := agg.sum / float64(agg.count)
		if stats.AccountMostRemaining == nil || avg > stats.AccountMostRemaining.RemainingFraction {
			stats.AccountMostRemaining = &domain.AccountRemaining{
				Email:             email,
				RemainingFraction: avg,
			}
		}
	}

	return stats, nil
}

func cycleKey(email, modelID, resetTime string) string {
	return email + "\x00" + modelID + "\x00" + resetTime
}

func scanFractionSamples(rows *sql.Rows) ([]domain.FractionSample, error) {
	var samples []domain.FractionSample
	for rows.Next() {
		var (
			sample     domain.FractionSample
			capturedAt string
		)
		if err := rows.Scan(&capturedAt, &sample.Label, &sample.ModelID, &sample.RemainingFraction); err != nil {
			return nil, err
		}
		t, err := parseDBTime(capturedAt, "captured_at")
		if err != nil {
			return nil, err
		}
		sample.CapturedAt = t
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

func (d *DB) getModelQuotas(snapshotID int64) ([]domain.ModelQuota, error) {
	rows, err := d.sql.Query(`
		SELECT label, model_id, remaining_fraction, remaining_pct,
		       is_exhausted, reset_time, pool_reset_time, time_until_reset_ms
		FROM model_quotas
		WHERE snapshot_id = ?
	`, snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []domain.ModelQuota
	for rows.Next() {
		var m domain.ModelQuota
		var resetStr, poolResetStr *string
		if err := rows.Scan(&m.Label, &m.ModelID, &m.RemainingFraction, &m.RemainingPct,
			&m.IsExhausted, &resetStr, &poolResetStr, &m.TimeUntilResetMs); err != nil {
			return nil, err
		}
		if resetStr != nil {
			t, err := parseDBTime(*resetStr, "reset_time")
			if err != nil {
				return nil, err
			}
			m.ResetTime = &t
		}
		if poolResetStr != nil {
			t, err := parseDBTime(*poolResetStr, "pool_reset_time")
			if err != nil {
				return nil, err
			}
			m.PoolResetTime = &t
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

func (d *DB) addColumnIfMissing(table, column, colType string) error {
	if !sqliteIdentifier.MatchString(table) || !sqliteIdentifier.MatchString(column) {
		return fmt.Errorf("invalid sqlite identifier %q.%q", table, column)
	}

	var count int
	err := d.sql.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("pragma_table_info(%s): %w", table, err)
	}
	if count == 0 {
		_, err = d.sql.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, colType))
		if err != nil {
			return fmt.Errorf("ALTER TABLE %s ADD COLUMN %s: %w", table, column, err)
		}
	}
	return nil
}

func parseDBTime(value, field string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s %q: %w", field, value, err)
	}
	return t, nil
}
