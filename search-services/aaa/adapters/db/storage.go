package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"yadro.com/course/aaa/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) AddUser(ctx context.Context, user core.User) error {
	query := `
			INSERT INTO users (login, pass_hash, is_admin)
			VALUES ($1, $2, $3)
	`
	_, err := db.conn.ExecContext(ctx, query, user.Login, user.PassHash, user.IsAdmin)
	if err != nil {
		db.log.Error("db.add error", "error", err)
		return err
	}
	return nil
}

func (db *DB) GetUserByLogin(ctx context.Context, login string) (*core.User, error) {
	query := `
				SELECT * FROM users
				WHERE login = $1
	`
	user := core.User{}
	if err := db.conn.GetContext(ctx, &user, query, login); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &core.User{}, core.ErrUserNotFound
		}
		db.log.Error("get user by login db error", "error", err)
		return &core.User{}, err
	}
	return &user, nil
}

func (db *DB) GetUserByID(ctx context.Context, id int64) (*core.User, error) {
	query := `
				SELECT * FROM users
				WHERE id = $1
	`
	user := core.User{}
	if err := db.conn.GetContext(ctx, &user, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &core.User{}, core.ErrUserNotFound
		}
		db.log.Error("get user by id db error", "error", err)
		return &core.User{}, err
	}
	return &user, nil
}

// COMICS_HISTORY

func (db *DB) AddComicView(ctx context.Context, userID int64, comicID int) error {
	query := `
		INSERT INTO comics_history (user_id, comic_id) 
		VALUES ($1, $2)
	`
	_, err := db.conn.ExecContext(ctx, query, userID, comicID)
	if err != nil {
		db.log.Error("add comic view db error", "user_id", userID, "comic_id", comicID, "error", err)
		return err
	}

	return nil
}

func (db *DB) GetRecentViews(ctx context.Context, userID int64, limit int) ([]int, error) {
	query := `
		SELECT comic_id 
		FROM comics_history 
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2
	`
	var comicIDs []int
	err := db.conn.SelectContext(ctx, &comicIDs, query, userID, limit)
	if err != nil {
		db.log.Error("get recent views db error", "user_id", userID, "limit", limit, "error", err)
		return nil, err
	}
	return comicIDs, nil
}

func (db *DB) HasViewedComic(ctx context.Context, userID int64, comicID int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM comics_history 
			WHERE user_id = $1 AND comic_id = $2
		)
	`
	var exists bool
	err := db.conn.GetContext(ctx, &exists, query, userID, comicID)
	if err != nil {
		db.log.Error("has viewed comic db error", "user_id", userID, "comic_id", comicID, "error", err)
		return false, err
	}
	return exists, nil
}

// SEARCH_HISTORY

func (db *DB) AddSearch(ctx context.Context, userID int64, query string) error {
	cleanedQuery := strings.TrimSpace(query)
	if cleanedQuery == "" {
		return errors.New("empty search query")
	}
	querySQL := `
		INSERT INTO search_history (user_id, query) 
		VALUES ($1, $2)
	`

	_, err := db.conn.ExecContext(ctx, querySQL, userID, cleanedQuery)
	if err != nil {
		db.log.Error("add search db error", "user_id", userID, "query", cleanedQuery, "error", err)
		return err
	}

	return nil
}

func (db *DB) GetRecentSearches(ctx context.Context, userID int64, limit int) ([]string, error) {
	query := `
		SELECT query 
		FROM search_history 
		WHERE user_id = $1
		ORDER BY id DESC  -- сортируем по id (автоинкремент)
		LIMIT $2
	`
	var queries []string
	err := db.conn.SelectContext(ctx, &queries, query, userID, limit)
	if err != nil {
		db.log.Error("get recent searches db error", "user_id", userID, "limit", limit, "error", err)
		return nil, err
	}
	return queries, nil
}
