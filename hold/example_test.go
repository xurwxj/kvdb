package hold_test

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/xurwxj/kvdb/hold"
)

type Item struct {
	ID       int
	Category string `holdIndex:"Category"`
	Created  time.Time
}

func Example() {
	data := []Item{
		Item{
			ID:       0,
			Category: "blue",
			Created:  time.Now().Add(-4 * time.Hour),
		},
		Item{
			ID:       1,
			Category: "red",
			Created:  time.Now().Add(-3 * time.Hour),
		},
		Item{
			ID:       2,
			Category: "blue",
			Created:  time.Now().Add(-2 * time.Hour),
		},
		Item{
			ID:       3,
			Category: "blue",
			Created:  time.Now().Add(-20 * time.Minute),
		},
	}

	dir := tempdir()
	defer os.RemoveAll(dir)

	options := hold.DefaultOptions
	options.Dir = dir
	options.ValueDir = dir
	store, err := hold.Open(options)
	defer store.Close()

	if err != nil {
		// handle error
		log.Fatal(err)
	}

	// insert the data in one transaction

	err = store.Badger().Update(func(tx *badger.Txn) error {
		for i := range data {
			err := store.TxInsert(tx, data[i].ID, data[i])
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		// handle error
		log.Fatal(err)
	}

	// Find all items in the blue category that have been created in the past hour
	var result []Item

	err = store.Find(&result, hold.Where("Category").Eq("blue").And("Created").Ge(time.Now().Add(-1*time.Hour)))

	if err != nil {
		// handle error
		log.Fatal(err)
	}

	fmt.Println(result[0].ID)
	// Output: 3

}
