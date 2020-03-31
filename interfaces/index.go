package interfaces

// OpSet identifier for set data into storeage
const OpSet = "set"

// OpDel identifier for delete data from storage
const OpDel = "del"

// Operation represents structure to set/del from storage
type Operation struct {
	Key   string
	Value []byte
	Op    string
}

// DbStorage represent base db storage interface
type DbStorage interface {
	Set(key string, value []byte) (err error)
	Del(key string) (err error)
	Get(key string) (value []byte, err error)
	Iterate(fn func(key []byte, value []byte))
	IterateByPrefix(prefix []byte, limit uint64, fn func(key []byte, value []byte)) uint64
	IterateByPrefixFrom(prefix []byte, from []byte, limit uint64, fn func(key []byte, value []byte)) uint64
	DeleteByPrefix(prefix []byte)
	KeysByPrefixCount(prefix []byte) uint64
	ProcessBatch(batch []*Operation) (err error)
	Close() error
}
