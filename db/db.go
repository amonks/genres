package db

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB represents our sqlite3 database file.
type DB struct {
	*gorm.DB
}

//go:embed schema.sql
var schema string

// Open returns a connection to a migrated sqlite3 database file on disk,
// creating the file and running migrations if necessary.
func Open() (*DB, error) {
	const filename = "genres.db"
	gdb, err := gorm.Open(sqlite.Open(filename), &gorm.Config{
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
		return nil, fmt.Errorf("error opening db file at '%s': %w", filename, err)
	}

	db := &DB{gdb}

	if err := db.Exec(schema).Error; err != nil {
		return nil, fmt.Errorf("error migrating db at '%s': %w", filename, err)
	}

	return db, nil
}

func (db *DB) Close() error {
	instance, err := db.DB.DB()
	if err != nil {
		fmt.Println("error getting db to close:", err)
		return err
	}
	return instance.Close()
}
