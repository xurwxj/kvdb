# kvdb
kvdb interface for badger etc.


## Indexes
Indexes allow you to skip checking any records that don't meet your index criteria.  If you have 1000 records and only
10 of them are of the Division you want to deal with, then you don't need to check to see if the other 990 records match
your query criteria if you create an index on the Division field.  The downside of an index is added disk reads and writes
on every write operation.  For read heavy operations datasets, indexes can be very useful.

In every Hold store, there will be a reserved bucket *_indexes* which will be used to hold indexes that point back
to another bucket's Key system.  Indexes will be defined by setting the `hold:"index"` struct tag on a field in a type.

```Go
type Person struct {
	Name string
	Division string `hold:"index"`
}

// alternate struct tag if you wish to specify the index name
type Person struct {
	Name string
	Division string `holdIndex:"IdxDivision"`
}

```

This means that there will be an index created for `Division` that will contain the set of unique divisions, and the
main record keys they refer to. 

Optionally, you can implement the `Storer` interface, to specify your own indexes, rather than using the `holdIndex`
struct tag.

## Queries
Queries are chain-able constructs that filters out any data that doesn't match it's criteria. An index will be used if
the `.Index()` chain is called, otherwise Hold won't use any index.

Queries will look like this:
```Go
s.Find(hold.Where("FieldName").Eq(value).And("AnotherField").Lt(AnotherValue).Or(hold.Where("FieldName").Eq(anotherValue)))

```

Fields must be exported, and thus always need to start with an upper-case letter.  Available operators include:
* Equal - `Where("field").Eq(value)`
* Not Equal - `Where("field").Ne(value)`
* Greater Than - `Where("field").Gt(value)`
* Less Than - `Where("field").Lt(value)`
* Less than or Equal To - `Where("field").Le(value)`
* Greater Than or Equal To - `Where("field").Ge(value)`
* In - `Where("field").In(val1, val2, val3)`
* IsNil - `Where("field").IsNil()`
* Regular Expression - `Where("field").RegExp(regexp.MustCompile("ea"))`
* Matches Function - `Where("field").MatchFunc(func(ra *RecordAccess) (bool, error))`
* Skip - `Where("field").Eq(value).Skip(10)`
* Limit - `Where("field").Eq(value).Limit(10)`
* SortBy - `Where("field").Eq(value).SortBy("field1", "field2")`
* Reverse - `Where("field").Eq(value).SortBy("field").Reverse()`
* Index - `Where("field").Eq(value).Index("indexName")`


If you want to run a query's criteria against the Key value, you can use the `hold.Key` constant:
```Go

store.Find(&result, hold.Where(hold.Key).Ne(value))

```

You can access nested structure fields in queries like this:

```Go
type Repo struct {
  Name string
  Contact ContactPerson
}

type ContactPerson struct {
  Name string
}

store.Find(&repo, hold.Where("Contact.Name").Eq("some-name")
```

Instead of passing in a specific value to compare against in a query, you can compare against another field in the same
struct.  Consider the following struct:

```Go
type Person struct {
	Name string
	Birth time.Time
	Death time.Time
}

```

If you wanted to find any invalid records where a Person's death was before their birth, you could do the following:

```Go

store.Find(&result, hold.Where("Death").Lt(hold.Field("Birth")))

```

Queries can be used in more than just selecting data.  You can delete or update data that matches a query.

Using the example above, if you wanted to remove all of the invalid records where Death < Birth:

```Go

// you must pass in a sample type, so Hold knows which bucket to use and what indexes to update
store.DeleteMatching(&Person{}, hold.Where("Death").Lt(hold.Field("Birth")))

```

Or if you wanted to update all the invalid records to flip/flop the Birth and Death dates:
```Go

store.UpdateMatching(&Person{}, hold.Where("Death").Lt(hold.Field("Birth")), func(record interface{}) error {
	update, ok := record.(*Person) // record will always be a pointer
	if !ok {
		return fmt.Errorf("Record isn't the correct type!  Wanted Person, got %T", record)
	}

	update.Birth, update.Death = update.Death, update.Birth

	return nil
})
```

### Keys in Structs

A common scenario is to store the hold Key in the same struct that is stored in the badgerDB value.  You can
automatically populate a record's Key in a struct by using the `hold:"key"` struct tag when running `Find` queries.

Another common scenario is to insert data with an auto-incrementing key assigned by the database.
When performing an `Insert`, if the type of the key matches the type of the `hold:"key"` tagged field,
the data is passed in by reference, **and** the field's current value is the zero-value for that type,
then it is set on the data _before_ insertion.

```Go
type Employee struct {
	ID uint64 `hold:"key"`
	FirstName string
	LastName string
	Division string
	Hired time.Time
}

// old struct tag, currenty still supported but may be deprecated in the future
type Employee struct {
	ID uint64 `holdKey`
	FirstName string
	LastName string
	Division string
	Hired time.Time
}
```
hold assumes only one of such struct tags exists. If a value already exists in the key field, it will be overwritten.

If you want to insert an auto-incrementing Key you can pass the `hold.NextSequence()` func as the Key value.

```Go
err := store.Insert(hold.NextSequence(), data)
```

The key value will be a `uint64`.

If you want to know the value of the auto-incrementing Key that was generated using `hold.NextSequence()`,
then make sure to pass your data by value and that the `holdKey` tagged field is of type `uint64`.

```Go
err := store.Insert(hold.NextSequence(), &data)
```


### Unique Constraints

You can create a unique constraint on a given field by using the `hold:"unique"` struct tag:

```Go
type User struct {
  Name string 
  Email string `hold:"unique"` // this field will be indexed with a unique constraint
}
```

The example above will only allow one record of type `User` to exist with a given `Email` field.  Any insert, update
or upsert that would violate that constraint will fail and return the `hold.ErrUniqueExists` error.


### Aggregate Queries

Aggregate queries are queries that group results by a field.  For example, lets say you had a collection of employees:

```Go
type Employee struct {
	FirstName string
	LastName string
	Division string
	Hired time.Time
}
```

And you wanted to find the most senior (first hired) employee in each division:

```Go

result, err := store.FindAggregate(&Employee{}, nil, "Division") //nil query matches against all records
```

This will return a slice of `Aggregate Result` from which you can extract your groups and find Min, Max, Avg, Count,
etc.

```Go
for i := range result {
	var division string
	employee := &Employee{}

	result[i].Group(&division)
	result[i].Min("Hired", employee)

	fmt.Printf("The most senior employee in the %s division is %s.\n",
		division, employee.FirstName + " " + employee.LastName)
}
```

Aggregate queries become especially powerful when combined with the sub-querying capability of `MatchFunc`.


Many more examples of queries can be found in the [find_test.go](https://github.com/timshannon/hold/blob/master/find_test.go)
file in this repository.

## Comparing

Just like with Go, types must be the same in order to be compared with each other.  You cannot compare an int to a int32.
The built-in Go comparable types (ints, floats, strings, etc) will work as expected.  Other types from the standard library
can also be compared such as `time.Time`, `big.Rat`, `big.Int`, and `big.Float`.  If there are other standard library
types that I missed, let me know.

You can compare any custom type either by using the `MatchFunc` criteria, or by satisfying the `Comparer` interface with
your type by adding the Compare method: `Compare(other interface{}) (int, error)`.

If a type doesn't have a predefined comparer, and doesn't satisfy the Comparer interface, then the types value is converted
to a string and compared lexicographically.

## Behavior Changes
Since Hold is a higher level interface than Badger DB, there are some added helpers.  Instead of *Put*, you
have the options of:
* *Insert* - Fails if key already exists.
* *Update* - Fails if key doesn't exist `ErrNotFound`.
* *Upsert* - If key doesn't exist, it inserts the data, otherwise it updates the existing record.

When getting data instead of returning `nil` if a value doesn't exist, Hold returns `hold.ErrNotFound`, and
similarly when deleting data, instead of silently continuing if a value isn't found to delete, Hold returns
`hold.ErrNotFound`.  The exception to this is when using query based functions such as `Find` (returns an empty slice),
`DeleteMatching` and `UpdateMatching` where no error is returned.


## When should I use Hold?
Hold will be useful in the same scenarios where BadgerDB is useful, with the added benefit of being able to retire
some of your data filtering code and possibly improved performance.

You can also use it instead of SQLite for many scenarios.  Hold's main benefit over SQLite is its simplicity when
working with Go Types.  There is no need for an ORM layer to translate records to types, simply put types in, and get
types out.  You also don't have to deal with database initialization.  Usually with SQLite you'll need several scripts
to create the database, create the tables you expect, and create any indexes.  With Hold you simply open a new file
and put any type of data you want in it.

```Go
store, err := hold.Open(filename, 0666, nil)
if err != nil {
	//handle error
}
err = store.Insert("key", &Item{
	Name:    "Test Name",
	Created: time.Now(),
})

```
