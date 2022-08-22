package migrate

import (
	"fmt"

	"github.com/go-pg/migrations"
)

func init() {
	migrations.Register(func(db migrations.DB) error {
		fmt.Println("alter table requests...")
		_, err := db.Exec(`alter table requests
    add address text;

create unique index requests_address_uindex
    on requests (address);
		`)
		return err
	},
		func(db migrations.DB) error {
			fmt.Println("reverting table requests...")
			_, err := db.Exec(`ALTER TABLE requests
									DROP COLUMN address RESTRICT;
		`)
			return err
		})
}
