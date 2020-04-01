package hold

import (
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
)

const (
	// HoldIndexTag is the struct tag used to define an a field as indexable for a hold
	HoldIndexTag = "holdIndex"

	// HoldKeyTag is the struct tag used to define an a field as a key for use in a Find query
	HoldKeyTag = "holdKey"

	// holdPrefixTag is the prefix for an alternate (more standard) version of a struct tag
	holdPrefixTag         = "hold"
	holdPrefixIndexValue  = "index"
	holdPrefixKeyValue    = "key"
	holdPrefixUniqueValue = "unique"
)

// Store is a hold wrapper around a badger DB
type Store struct {
	db               *badger.DB
	sequenceBandwith uint64
	sequences        *sync.Map
}

// Options allows you set different options from the defaults
// For example the encoding and decoding funcs which default to Gob
type Options struct {
	Encoder          EncodeFunc
	Decoder          DecodeFunc
	SequenceBandwith uint64
	badger.Options
}

// DefaultOptions are a default set of options for opening a Hold database
// Includes badgers own default options
var DefaultOptions = Options{
	Options:          badger.DefaultOptions(""),
	Encoder:          DefaultEncode,
	Decoder:          DefaultDecode,
	SequenceBandwith: 100,
}

// Open opens or creates a hold file.
func Open(options Options) (*Store, error) {

	encode = options.Encoder
	decode = options.Decoder

	db, err := badger.Open(options.Options)
	if err != nil {
		return nil, err
	}

	go runStorageGC(db)

	return &Store{
		db:               db,
		sequenceBandwith: options.SequenceBandwith,
		sequences:        &sync.Map{},
	}, nil
}

func runStorageGC(db *badger.DB) {
	timer := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-timer.C:
			storageGC(db)
		}
	}
}

func storageGC(db *badger.DB) {
again:
	err := db.RunValueLogGC(0.5)
	if err == nil {
		goto again
	}
}

// Badger returns the underlying Badger DB the hold is based on
func (s *Store) Badger() *badger.DB {
	return s.db
}

// Close closes the badger db
func (s *Store) Close() error {
	var err error
	s.sequences.Range(func(key, value interface{}) bool {
		err = value.(*badger.Sequence).Release()
		if err != nil {
			return false
		}
		return true
	})
	if err != nil {
		return err
	}
	return s.db.Close()
}

/*
	NOTE: Not going to implement ReIndex and Remove index
	I had originally created these to make the transition from a plain bolt or badger DB easier
	but there is too much chance for lost data, and it's probably better that any conversion be
	done by the developer so they can directly manage how they want data to be migrated.
	If you disagree, feel free to open an issue and we can revisit this.
*/

// Storer is the Interface to implement to skip reflect calls on all data passed into the hold
type Storer interface {
	Type() string              // used as the badgerdb index prefix
	Indexes() map[string]Index //[indexname]indexFunc
}

// anonType is created from a reflection of an unknown interface
type anonStorer struct {
	rType   reflect.Type
	indexes map[string]Index
}

// Type returns the name of the type as determined from the reflect package
func (t *anonStorer) Type() string {
	return t.rType.Name()
}

// Indexes returns the Indexes determined by the reflect package on this type
func (t *anonStorer) Indexes() map[string]Index {
	return t.indexes
}

// newStorer creates a type which satisfies the Storer interface based on reflection of the passed in dataType
// if the Type doesn't meet the requirements of a Storer (i.e. doesn't have a name) it panics
// You can avoid any reflection costs, by implementing the Storer interface on a type
func newStorer(dataType interface{}) Storer {
	s, ok := dataType.(Storer)

	if ok {
		return s
	}

	tp := reflect.TypeOf(dataType)

	for tp.Kind() == reflect.Ptr {
		tp = tp.Elem()
	}

	storer := &anonStorer{
		rType:   tp,
		indexes: make(map[string]Index),
	}

	if storer.rType.Name() == "" {
		panic("Invalid Type for Storer.  Type is unnamed")
	}

	if storer.rType.Kind() != reflect.Struct {
		panic("Invalid Type for Storer.  Hold only works with structs")
	}

	for i := 0; i < storer.rType.NumField(); i++ {

		indexName := ""
		unique := false

		if strings.Contains(string(storer.rType.Field(i).Tag), HoldIndexTag) {
			indexName = storer.rType.Field(i).Tag.Get(HoldIndexTag)

			if indexName != "" {
				indexName = storer.rType.Field(i).Name
			}
		} else if tag := storer.rType.Field(i).Tag.Get(holdPrefixTag); tag != "" {
			if tag == holdPrefixIndexValue {
				indexName = storer.rType.Field(i).Name
			} else if tag == holdPrefixUniqueValue {
				indexName = storer.rType.Field(i).Name
				unique = true
			}
		}

		if indexName != "" {
			storer.indexes[indexName] = Index{
				IndexFunc: func(name string, value interface{}) ([]byte, error) {
					tp := reflect.ValueOf(value)
					for tp.Kind() == reflect.Ptr {
						tp = tp.Elem()
					}

					return encode(tp.FieldByName(name).Interface())
				},
				Unique: unique,
			}
		}
	}

	return storer
}

func (s *Store) getSequence(typeName string) (uint64, error) {
	seq, ok := s.sequences.Load(typeName)
	if !ok {
		newSeq, err := s.Badger().GetSequence([]byte(typeName), s.sequenceBandwith)
		if err != nil {
			return 0, err
		}
		s.sequences.Store(typeName, newSeq)
		seq = newSeq
	}

	return seq.(*badger.Sequence).Next()
}

func typePrefix(typeName string) []byte {
	return []byte("bh_" + typeName)
}
