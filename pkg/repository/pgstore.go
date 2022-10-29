package repository

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

//go:embed migrations
var migrations embed.FS

type User struct {
	ID       int    `json:"id"       db:"id"`
	Username string `json:"username" db:"username"`
	Password string `json:"password" db:"password"`
}

type PG struct {
	log *logrus.Entry
	db  *sqlx.DB
	dsn string
}

var (
	ErrInvalidCredentials = fmt.Errorf("err invalid credintials")
	ErrAlreadyExists      = fmt.Errorf("username alredy exists")
)

func NewRepo(ctx context.Context, log *logrus.Logger, dsn string) (*PG, error) {
	db, err := sqlx.ConnectContext(ctx, "pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("err connecting to PG : %w", err)
	}
	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("err pinging pg after initing connection: %w", err)
	}
	pg := &PG{
		log: log.WithField("component", "pgstore"),
		db:  db,
		dsn: dsn,
	}
	return pg, nil
}

func (pg *PG) Migrate(direction migrate.MigrationDirection) error {
	conn, err := sql.Open("pgx", pg.dsn)
	if err != nil {
		return err
	}
	defer func() {
		if err = conn.Close(); err != nil {
			pg.log.Error("err closing migration connection")
		}
	}()
	assetDir := func() func(string) ([]string, error) {
		return func(path string) ([]string, error) {
			dirEntry, er := migrations.ReadDir(path)
			if er != nil {
				return nil, er
			}
			entries := make([]string, 0)
			for _, e := range dirEntry {
				entries = append(entries, e.Name())
			}

			return entries, nil
		}
	}()
	asset := migrate.AssetMigrationSource{
		Asset:    migrations.ReadFile,
		AssetDir: assetDir,
		Dir:      "migrations",
	}
	_, err = migrate.Exec(conn, "postgres", asset, direction)
	return err
}

func (pg *PG) Close() {
	if err := pg.db.Close(); err != nil {
		pg.log.Errorf("err closing pg connection: %v", err)
	}
}

func (pg *PG) hashPassword(password string) []byte {
	cost := 10
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), cost)
	return hash
}

func (pg *PG) Registration(ctx context.Context, user User) (int, error) {
	query := `INSERT INTO users (username, password) VALUES ($1,$2) RETURNING id`
	var id int
	row := pg.db.QueryRowContext(ctx, query, user.Username, pg.hashPassword(user.Password))
	var pgErr *pgconn.PgError
	if err := row.Scan(&id); err != nil {
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				return 0, ErrAlreadyExists
			}
		}
		return 0, fmt.Errorf("err while the registration: %w", err)
	}
	return id, nil
}

func (pg *PG) Login(ctx context.Context, user User) (User, error) {
	var hash string
	query := `SELECT password FROM users WHERE username = $1`
	err := pg.db.GetContext(ctx, &hash, query, user.Username)
	if err != nil {
		return User{}, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(user.Password))
	if err != nil {
		return User{}, ErrInvalidCredentials
	}

	query = `SELECT id, username, password FROM users WHERE username = $1`
	var resp User
	if err = pg.db.GetContext(ctx, &resp, query, user.Username); err != nil {
		return User{}, fmt.Errorf("err logining user : %w", err)
	}
	return resp, nil
}
