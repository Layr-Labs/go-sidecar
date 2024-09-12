package sqlite

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"

	goSqlite "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// bytesToHex is a custom SQLite function that converts a JSON byte array to a hex string.
//
// @param jsonByteArray: a JSON byte array, e.g. [1, 2, 3, ...]
// @return: a hex string without a leading 0x, e.g. 78cc56f0700e7ba5055f12...
func bytesToHex(jsonByteArray string) (string, error) {
	jsonBytes := make([]byte, 0)
	err := json.Unmarshal([]byte(jsonByteArray), &jsonBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(jsonBytes), nil
}

func NewSqlite(path string) gorm.Dialector {
	sql.Register("sqlite3_with_extensions", &goSqlite.SQLiteDriver{
		ConnectHook: func(conn *goSqlite.SQLiteConn) error {
			return conn.RegisterFunc("bytes_to_hex", bytesToHex, true)
		},
	})
	return &sqlite.Dialector{
		DriverName: "sqlite3_with_extensions",
		DSN:        path,
	}
}

func NewGormSqliteFromSqlite(sqlite gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = normal;`,
		`pragma mmap_size = 30000000000;`,
	}

	for _, pragma := range pragmas {
		res := db.Exec(pragma)
		if res.Error != nil {
			return nil, res.Error
		}
	}

	return db, nil
}

func WrapTxAndCommit[T any](fn func(*gorm.DB) (T, error), db *gorm.DB, tx *gorm.DB) (T, error) {
	exists := tx != nil

	if !exists {
		tx = db.Begin()
	}

	res, err := fn(tx)

	if err != nil && !exists {
		fmt.Printf("Rollback transaction\n")
		tx.Rollback()
	}
	if err == nil && !exists {
		tx.Commit()
	}
	return res, err
}

func IsDuplicateKeyError(err error) bool {
	r := regexp.MustCompile(`UNIQUE constraint failed`)

	return r.MatchString(err.Error())
}
