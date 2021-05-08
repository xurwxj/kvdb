package hold_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/xurwxj/kvdb/hold"
)

func TestOpen(t *testing.T) {
	opt := testOptions()
	store, err := hold.Open(opt)
	if err != nil {
		t.Fatalf("Error opening %s: %s", opt.Dir, err)
	}

	if store == nil {
		t.Fatalf("store is null!")
	}

	err = store.Close()
	if err != nil {
		t.Fatal(err)
	}
	err = os.RemoveAll(opt.Dir)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBadger(t *testing.T) {
	testWrap(t, func(store *hold.Store, t *testing.T) {
		b := store.Badger()
		if b == nil {
			t.Fatalf("Badger is null in hold")
		}
	})
}

func TestAlternateEncoding(t *testing.T) {
	opt := testOptions()
	opt.Encoder = json.Marshal
	opt.Decoder = json.Unmarshal
	store, err := hold.Open(opt)

	if err != nil {
		t.Fatalf("Error opening %s: %s", opt.Dir, err)
	}

	defer os.RemoveAll(opt.Dir)
	defer store.Close()

	insertTestData(t, store)

	tData := testData[3]

	var result []ItemTest

	store.Find(&result, hold.Where(hold.Key).Eq(tData.Key))

	if len(result) != 1 {
		if testing.Verbose() {
			t.Fatalf("Find result count is %d wanted %d.  Results: %v", len(result), 1, result)
		}
		t.Fatalf("Find result count is %d wanted %d.", len(result), 1)
	}

	if !result[0].equal(&tData) {
		t.Fatalf("Results not equal! Wanted %v, got %v", tData, result[0])
	}

}

func TestGetUnknownType(t *testing.T) {
	opt := testOptions()
	store, err := hold.Open(opt)
	if err != nil {
		t.Fatalf("Error opening %s: %s", opt.Dir, err)
	}

	defer os.RemoveAll(opt.Dir)
	defer store.Close()

	type test struct {
		Test string
	}

	var result test
	err = store.Get("unknownKey", &result)
	if err != hold.ErrNotFound {
		t.Errorf("Expected error of type ErrNotFound, not %T", err)
	}
}

// utilities
func testWrap(t *testing.T, tests func(store *hold.Store, t *testing.T)) {
	opt := testOptions()
	var err error
	store, err := hold.Open(opt)
	if err != nil {
		t.Fatalf("Error opening %s: %s", opt.Dir, err)
	}

	if store == nil {
		t.Fatalf("store is null!")
	}

	tests(store, t)
	store.Close()
	os.RemoveAll(opt.Dir)
}

type emptyLogger struct{}

func (e emptyLogger) Errorf(msg string, data ...interface{})   {}
func (e emptyLogger) Infof(msg string, data ...interface{})    {}
func (e emptyLogger) Warningf(msg string, data ...interface{}) {}
func (e emptyLogger) Debugf(msg string, data ...interface{})   {}

func testOptions() hold.Options {
	opt := hold.DefaultOptions
	opt.Dir = tempdir()
	opt.ValueDir = opt.Dir
	opt.Logger = emptyLogger{}
	// opt.ValueLogLoadingMode = options.FileIO // slower but less memory usage
	// opt.TableLoadingMode = options.FileIO
	// opt.NumMemtables = 1
	// opt.NumLevelZeroTables = 1
	// opt.NumLevelZeroTablesStall = 2
	// opt.NumCompactors = 1
	return opt
}

// tempdir returns a temporary dir path.
func tempdir() string {
	name, err := ioutil.TempDir("", "hold-")
	if err != nil {
		panic(err)
	}
	return name
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
