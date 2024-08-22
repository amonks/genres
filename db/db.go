package db

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB represents our sqlite3 database file.
type DB struct {
	rw *gorm.DB
	ro *gorm.DB

	wmu sync.Mutex
}

func (db *DB) hold() func() {
	db.wmu.Lock()
	return func() { db.wmu.Unlock() }
}

//go:embed schema.sql
var schema string

// Open returns a connection to a migrated sqlite3 database file on disk,
// creating the file and running migrations if necessary.
func Open(filename string) (*DB, error) {
	rodb, err := openDB(filename, false)
	if err != nil {
		return nil, err
	}

	rwdb, err := openDB(filename, true)
	if err != nil {
		return nil, err
	}

	db := &DB{
		ro: rodb,
		rw: rwdb,
	}

	if err := db.rw.Exec(schema).Error; err != nil {
		return nil, fmt.Errorf("error migrating db at '%s': %w", filename, err)
	}

	return db, nil
}

func openDB(filename string, rw bool) (*gorm.DB, error) {
	options := "mode=ro&_query_only=true"
	if rw {
		options = "mode=rw&_query_only=false&_txlock=immediate"
	}
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("%s?_journal_mode=wal&_foreign_keys=true&_synchronous=1&%s", filename, options)), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stderr, "\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				ParameterizedQueries:      true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		return nil, err
	}
	if rw {
		rawdb, err := db.DB()
		if err != nil {
			return nil, err
		}
		rawdb.SetMaxOpenConns(1)
	}
	return db, nil
}

func (db *DB) Close() error {
	defer db.hold()()
	roinstance, err := db.ro.DB()
	if err != nil {
		fmt.Println("error getting db to close:", err)
		return err
	}
	if err := roinstance.Close(); err != nil {
		return err
	}

	rwinstance, err := db.rw.DB()
	if err != nil {
		fmt.Println("error getting db to close:", err)
		return err
	}
	if err := rwinstance.Close(); err != nil {
		return err
	}

	return nil
}
