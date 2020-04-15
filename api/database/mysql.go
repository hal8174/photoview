package database

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/pkg/errors"

	// Load mysql driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/mysql"

	// Migrate from file
	_ "github.com/golang-migrate/migrate/source/file"
)

// SetupDatabase connects to the database using environment variables
func SetupDatabase() (*sql.DB, error) {

	address, err := url.Parse(os.Getenv("MYSQL_URL"))
	if err != nil {
		return nil, errors.Wrapf(err, "Could not parse mysql url")
	}

	if address.String() == "" {
		return nil, errors.New("Environment variable MYSQL_URL missing, exiting")
	}

	queryValues := address.Query()
	queryValues.Add("multiStatements", "true")
	queryValues.Add("parseTime", "true")

	address.RawQuery = queryValues.Encode()

	log.Printf("Connecting to database: %s", address)

	var db *sql.DB
	tryCount := 0

	for {
		db, err = sql.Open("mysql", address.String())
		if err != nil {
			if tryCount < 4 {
				tryCount++
				log.Printf("WARN: Could not connect to database: %s. Will retry after 1 second", err.Error())
				time.Sleep(time.Second)
				continue
			} else {
				return nil, errors.New("Could not connect to database, exiting")
			}
		}

		break
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Wrap(err, "Could not ping database, exiting")
	}

	db.SetMaxOpenConns(80)

	return db, nil
}

func MigrateDatabase(db *sql.DB) error {
	driver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://database/migrations",
		"mysql",
		driver,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		if err.Error() == "no change" {
			log.Println("Database is up to date")
		} else {
			return err
		}
	} else {
		log.Println("Database migrated")
	}

	return nil
}
