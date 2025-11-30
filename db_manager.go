package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

type ctxKey string

const readYourWritesKey ctxKey = "readYourWrites"

func EnableReadYourWrites(ctx context.Context) context.Context {
	return context.WithValue(ctx, readYourWritesKey, true)
}

type DBManager struct {
	writeDB *sql.DB
	readDB  *sql.DB

	followerHealthy atomic.Bool

	leaderReads   int64
	followerReads int64
	fallbackReads int64
	writes        int64
}

type DBStats struct {
	LeaderReads     int64
	FollowerReads   int64
	FallbackReads   int64
	Writes          int64
	FollowerHealthy bool
}

func (d *DBManager) GetStats() DBStats {
	return DBStats{
		LeaderReads:     atomic.LoadInt64(&d.leaderReads),
		FollowerReads:   atomic.LoadInt64(&d.followerReads),
		FallbackReads:   atomic.LoadInt64(&d.fallbackReads),
		Writes:          atomic.LoadInt64(&d.writes),
		FollowerHealthy: d.isFollowerHealthy(),
	}
}

func (d *DBManager) monitorFollowerHealth() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	prevState := d.followerHealthy.Load()

	for range ticker.C {
		// If the follower was never connected, skip ping
		if d.readDB == nil {
			continue
		}

		err := d.readDB.Ping()
		currentState := err == nil

		// Update atomic flag
		d.followerHealthy.Store(currentState)

		// Log on transitions only
		if currentState != prevState {
			if currentState {
				log.Println("[DB] Follower is UP (Ping OK)")
			} else {
				log.Println("[DB] Follower is DOWN (Ping failed)")
			}
		}

		prevState = currentState
	}
}

func NewDBManager(writeDSN, readDSN string) (*DBManager, error) {
	dbManger := &DBManager{}
	dbManger.followerHealthy.Store(false)

	leader, err := sql.Open("postgres", writeDSN)
	if err != nil {
		return nil, fmt.Errorf("could not open leader db: %w", err)
	}
	leader.SetMaxOpenConns(100)
	leader.SetMaxIdleConns(5)
	leader.SetConnMaxLifetime(5 * time.Minute)
	dbManger.writeDB = leader

	// Connect to follower (reads)
	follower, err := sql.Open("postgres", readDSN)
	if err == nil {
		if err := follower.Ping(); err == nil {
			follower.SetMaxOpenConns(100)
			follower.SetMaxIdleConns(5)
			follower.SetConnMaxLifetime(5 * time.Minute) // Add this

			dbManger.readDB = follower
			dbManger.followerHealthy.Store(true)
			log.Println("Follower connected & healthy")
		} else {
			log.Println("Follower unreachable on Ping()")
		}
	} else {
		log.Println("Follower DSN invalid")
	}

	go dbManger.monitorFollowerHealth()

	return dbManger, err
}

func (d *DBManager) Close() error {
	var errLeader, errFollower error
	if d.writeDB != nil {
		errLeader = d.writeDB.Close()
	}
	if d.readDB != nil {
		errFollower = d.readDB.Close()
	}
	if errLeader != nil {
		return errLeader
	}
	return errFollower
}

// DB Manager methods
func (d *DBManager) isFollowerHealthy() bool {
	return d.followerHealthy.Load() && d.readDB != nil
}

func (d *DBManager) shouldUseLeader(ctx context.Context) bool {
	v := ctx.Value(readYourWritesKey)
	b, ok := v.(bool)
	return ok && b
}

func (d *DBManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	atomic.AddInt64(&d.writes, 1) // Count as a write operation
	return d.writeDB.BeginTx(ctx, opts)
}

func (d *DBManager) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	atomic.AddInt64(&d.writes, 1)
	return d.writeDB.ExecContext(ctx, query, args...)
}

func (d *DBManager) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	if d.shouldUseLeader(ctx) {
		atomic.AddInt64(&d.leaderReads, 1)
		return d.writeDB.QueryContext(ctx, query, args...)
	}

	if d.isFollowerHealthy() {
		rows, err := d.readDB.QueryContext(ctx, query, args...)
		if err == nil {
			atomic.AddInt64(&d.followerReads, 1)
			return rows, nil
		}
		log.Println("Follower failed, falling back to leader")
		atomic.AddInt64(&d.fallbackReads, 1)
	}

	atomic.AddInt64(&d.leaderReads, 1)
	return d.writeDB.QueryContext(ctx, query, args...)

}

func (d *DBManager) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}

	if d.shouldUseLeader(ctx) {
		atomic.AddInt64(&d.leaderReads, 1)
		return d.writeDB.QueryRowContext(ctx, query, args...)
	}

	if d.isFollowerHealthy() {
		atomic.AddInt64(&d.followerReads, 1)
		return d.readDB.QueryRowContext(ctx, query, args...)
	}

	// Follower down, use leader
	atomic.AddInt64(&d.leaderReads, 1)
	return d.writeDB.QueryRowContext(ctx, query, args...)
}
