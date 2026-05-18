package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/update/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	dbInstance := &DB{
		log:  log,
		conn: conn,
	}

	return dbInstance, nil
}

func (db *DB) Close() error {
	err := db.conn.Close()
	if err != nil {
		db.log.Error("failed to close db connection", "error", err)
		return err
	}

	return nil
}

func (db *DB) Add(ctx context.Context, comics core.Comics) error {
	query := "INSERT INTO comics (id, url, words) VALUES($1, $2, $3)"

	_, err := db.conn.ExecContext(ctx, query, comics.ID, comics.URL, comics.Words)
	if err != nil {
		db.log.Error("failed to insert comics", "id", comics.ID, "url", comics.URL, "error", err)
		return err
	}

	return nil
}

func (db *DB) Stats(ctx context.Context) (core.DBStats, error) {
	var stats core.DBStats

	comicsCountQuery := "SELECT COUNT(*) FROM comics"
	err := db.conn.GetContext(ctx, &stats.ComicsFetched, comicsCountQuery)
	if err != nil {
		db.log.Error("failed to get comics count", "error", err)
		return core.DBStats{}, err
	}

	wordsTotalQuery := "SELECT coalesce(SUM(array_length(words,1)), 0) FROM comics"
	err = db.conn.GetContext(ctx, &stats.WordsTotal, wordsTotalQuery)
	if err != nil {
		db.log.Error("failed to get total words", "error", err)
		return core.DBStats{}, err
	}

	wordsUniqueQuery := "SELECT count(*) FROM (SELECT distinct(unnest(words)) FROM comics)"
	err = db.conn.GetContext(ctx, &stats.WordsUnique, wordsUniqueQuery)
	if err != nil {
		db.log.Error("failed to get unique words", "error", err)
		return core.DBStats{}, err
	}

	return stats, nil
}

func (db *DB) IDs(ctx context.Context) ([]int, error) {
	ids := make([]int, 0)

	query := "SELECT id FROM comics"
	err := db.conn.SelectContext(ctx, &ids, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		db.log.Error("failed to get comics ids", "error", err)
		return nil, err
	}

	return ids, nil
}

func (db *DB) Drop(ctx context.Context) error {
	query := "TRUNCATE comics"

	_, err := db.conn.ExecContext(ctx, query)
	if err != nil {
		db.log.Error("failed to truncate comics table", "error", err)
		return err
	}

	return nil
}
