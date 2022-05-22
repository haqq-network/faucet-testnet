package migrate

import (
	"fmt"

	"github.com/go-pg/migrations"
)

const requestsTable = `
create table requests
(
    id           serial,
    github       varchar,
    request_date bigint
);

create unique index requests_github_uindex
    on requests (github);

`

func init() {
	up := []string{
		requestsTable,
	}

	down := []string{
		`DROP TABLE requests`,
	}

	migrations.Register(func(db migrations.DB) error {
		fmt.Println("creating initial tables")
		for _, q := range up {
			_, err := db.Exec(q)
			if err != nil {
				return err
			}
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("dropping initial tables")
		for _, q := range down {
			_, err := db.Exec(q)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
