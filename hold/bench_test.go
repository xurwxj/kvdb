package hold_test

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger/v3"
	"github.com/xurwxj/kvdb/hold"
)

type BenchData struct {
	ID       int
	Category string
}

type BenchDataIndexed struct {
	ID       int
	Category string `holdIndex:"Category"`
}

var benchItem = BenchData{
	ID:       30,
	Category: "test category",
}

var benchItemIndexed = BenchData{
	ID:       30,
	Category: "test category",
}

// benchWrap creates a temporary database for testing and closes and cleans it up when
// completed.
func benchWrap(b *testing.B, options *hold.Options, bench func(store *hold.Store, b *testing.B)) {
	tempDir, err := ioutil.TempDir("", "hold_tests")
	if err != nil {
		b.Fatalf("Error opening %s: %s", tempDir, err)
	}
	defer os.RemoveAll(tempDir)

	if options == nil {
		options = &hold.DefaultOptions
	}

	options.Dir = tempDir
	options.ValueDir = tempDir

	store, err := hold.Open(*options)
	if err != nil {
		b.Fatalf("Error opening %s: %s", tempDir, err)
	}

	defer store.Close()
	if store == nil {
		b.Fatalf("store is null!")
	}

	bench(store, b)
}

var idVal uint64

func id() []byte {
	idVal++
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, idVal)
	return b
}

func BenchmarkRawInsert(b *testing.B) {
	data, err := hold.DefaultEncode(benchItem)
	if err != nil {
		b.Fatalf("Error encoding data for raw benchmarking: %s", err)
	}

	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {

			err = store.Badger().Update(func(tx *badger.Txn) error {
				return tx.Set(id(), data)
			})
			if err != nil {
				b.Fatalf("Error inserting raw bytes: %s", err)
			}
		}
	})
}

func BenchmarkNoIndexInsert(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := store.Insert(id(), benchItem)
			if err != nil {
				b.Fatalf("Error inserting into store: %s", err)
			}
		}
	})
}

func BenchmarkIndexedInsert(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := store.Insert(id(), benchItemIndexed)
			if err != nil {
				b.Fatalf("Error inserting into store: %s", err)
			}
		}
	})
}

func BenchmarkNoIndexUpsert(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := store.Upsert(id(), benchItem)
			if err != nil {
				b.Fatalf("Error inserting into store: %s", err)
			}
		}
	})
}

func BenchmarkIndexedUpsert(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := store.Upsert(id(), benchItemIndexed)
			if err != nil {
				b.Fatalf("Error inserting into store: %s", err)
			}
		}
	})
}

func BenchmarkNoIndexInsertJSON(b *testing.B) {
	opt := hold.DefaultOptions
	opt.Encoder = json.Marshal
	opt.Decoder = json.Unmarshal
	benchWrap(b, &opt, func(store *hold.Store, b *testing.B) {
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			err := store.Insert(id(), benchItem)
			if err != nil {
				b.Fatalf("Error inserting into store: %s", err)
			}
		}
	})
}

func BenchmarkFindNoIndex(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		for i := 0; i < 3; i++ {
			for k := 0; k < 100; k++ {
				err := store.Insert(id(), benchItem)
				if err != nil {
					b.Fatalf("Error inserting benchmarking data: %s", err)
				}
			}
			err := store.Insert(id(), &BenchData{
				ID:       30,
				Category: "findCategory",
			})
			if err != nil {
				b.Fatalf("Error inserting benchmarking data: %s", err)
			}

		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var result []BenchData

			err := store.Find(&result, hold.Where("Category").Eq("findCategory"))
			if err != nil {
				b.Fatalf("Error finding data in store: %s", err)
			}
		}
	})
}

func BenchmarkFindIndexed(b *testing.B) {
	benchWrap(b, nil, func(store *hold.Store, b *testing.B) {
		for i := 0; i < 3; i++ {
			for k := 0; k < 100; k++ {
				err := store.Insert(id(), benchItemIndexed)
				if err != nil {
					b.Fatalf("Error inserting benchmarking data: %s", err)
				}
			}
			err := store.Insert(id(), &BenchDataIndexed{
				ID:       30,
				Category: "findCategory",
			})
			if err != nil {
				b.Fatalf("Error inserting benchmarking data: %s", err)
			}

		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var result []BenchDataIndexed

			err := store.Find(&result, hold.Where("Category").Eq("findCategory"))
			if err != nil {
				b.Fatalf("Error finding data in store: %s", err)
			}
		}
	})
}
