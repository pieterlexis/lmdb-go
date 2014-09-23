package lmdb_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/bmatsuo/lmdb.exp"
)

// This complete example demonstrates the DupFixed (and DupSort) DBI flags.
func Example_dupFixed() {
	env, err := lmdb.NewEnv()
	if err != nil {
		log.Panic(err)
	}
	path, err := ioutil.TempDir("", "mdb_test")
	if err != nil {
		log.Panic(err)
	}
	defer os.RemoveAll(path)
	err = env.SetMaxDBs(1)
	if err != nil {
		log.Panic(err)
	}
	err = env.Open(path, 0, 0644)
	defer env.Close()
	if err != nil {
		log.Panic(err)
	}

	// open the database of friends' phone numbers.  a single person can have
	// multiple phone numbers and in this limited world phone nubers are all
	// the same length.
	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		log.Panic(err)
	}
	phonedb, err := txn.OpenDBI("phone-numbers", lmdb.Create|lmdb.DupSort|lmdb.DupFixed)
	if err != nil {
		txn.Abort()
		log.Panic(err)
	}
	cur, err := txn.OpenCursor(phonedb)
	if err != nil {
		txn.Abort()
		log.Panic(err)
	}
	entries := []struct{ name, number string }{
		{"alice", "234-1234"},
		{"bob", "825-1234"},
		{"carol", "824-1234"},
		{"carol", "828-1234"}, // DupSort stores multiple values for a key.
		{"carol", "502-1234"}, // DupSort values are stored in sorted order.
		{"jenny", "867-5309"},
	}
	for _, e := range entries {
		err = cur.Put([]byte(e.name), []byte(e.number), 0)
		if err != nil {
			txn.Abort()
			log.Panic(err)
		}
	}
	errc := cur.Close()
	if err == nil {
		err = errc
	}
	if err == nil {
		err = txn.Commit()
	} else {
		txn.Abort()
	}
	if err != nil {
		log.Panic(err)
	}

	// iterate the database and print the first two phone numbers for each
	// person.  this is similar to iterating a database normally but the
	// NextNoDup flag may be used to skip ahead.
	var lastk []byte
	var isdup bool
	txn, err = env.BeginTxn(nil, lmdb.Readonly)
	if err != nil {
		log.Panic(err)
	}
	defer txn.Abort()
	cur, err = txn.OpenCursor(phonedb)
	if err != nil {
		txn.Abort()
		log.Panic(err)
	}
	k, v, err := cur.Get(nil, nil, lmdb.First)
	for err != lmdb.ErrNotFound {
		if err != nil {
			log.Panic(err)
		}
		k, v, err = cur.Get(nil, nil, lmdb.GetCurrent)
		if err != nil {
			log.Panic(err)
		}
		isdup, lastk = bytes.Equal(lastk, k), k

		if !isdup {
			fmt.Printf("%s %s\n", k, v)
			k, v, err = cur.Get(k, v, lmdb.Next)
		} else {
			// print space instead of the name because it's on the previous
			// line and jump to the next key.
			fmt.Printf("%s %s\n", strings.Repeat(" ", len(k)), v)
			k, v, err = cur.Get(k, v, lmdb.NextNoDup)
		}
	}

	// Output:
	// alice 234-1234
	// bob 825-1234
	// carol 502-1234
	//       824-1234
	// jenny 867-5309
}

// This example shows how to use the Env type and open a database.
func ExampleEnv() {
	// create a directory to hold the database
	path, _ := ioutil.TempDir("", "mdb_test")
	defer os.RemoveAll(path)

	// open the LMDB environment
	env, err := lmdb.NewEnv()
	if err != nil {
		panic(err)
	}
	env.SetMaxDBs(1)
	env.Open(path, 0, 0664)
	defer env.Close()

	// open a database, creating it if necessary.
	txn, err := env.BeginTxn(nil, 0)
	if err != nil {
		panic(err)
	}
	db, err := txn.OpenDBI("exampledb", lmdb.Create)
	if err != nil {
		txn.Abort()
		panic(err)
	}

	// get statistics about the db. print the number of key-value pairs.
	stat, err := txn.Stat(db)
	if err != nil {
		txn.Abort()
		panic(err)
	}
	fmt.Println(stat.Entries)

	err = txn.Commit()
	if err != nil {
		panic(err)
	}

	// .. open more transactions and use the database

	// Output:
	// 0
}

// This example shows how to read and write data with a Txn.  Errors are
// ignored for brevity.  Real code should check and handle are errors which may
// require more modular code.
func ExampleTxn() {
	// create a directory to hold the database
	path, _ := ioutil.TempDir("", "mdb_test")
	defer os.RemoveAll(path)

	// open the LMDB environment
	env, _ := lmdb.NewEnv()
	env.SetMaxDBs(1)
	env.Open(path, 0, 0664)
	defer env.Close()

	// open a database.
	txn, _ := env.BeginTxn(nil, 0)
	db, _ := txn.OpenDBI("exampledb", lmdb.Create)
	txn.Commit()

	// write some data
	txn, _ = env.BeginTxn(nil, 0)
	txn.Put(db, []byte("key0"), []byte("val0"), 0)
	txn.Put(db, []byte("key1"), []byte("val1"), 0)
	txn.Put(db, []byte("key2"), []byte("val2"), 0)

	// inspect the transaction
	stat, _ := txn.Stat(db)
	fmt.Println(stat.Entries)

	// commit the transaction
	_ = txn.Commit()

	// perform random access on db.  Transactions created with the
	// lmdb.Readonly flag can always be aborted.
	txn, _ = env.BeginTxn(nil, lmdb.Readonly)
	defer txn.Abort()
	bval, _ := txn.Get(db, []byte("key1"))
	fmt.Println(string(bval))

	// Output:
	// 3
	// val1
}

// This example shows how to read and write data using a Cursor.  Errors are
// ignored for brevity.  Real code should check and handle are errors which may
// require more modular code.
func ExampleCursor() {
	// create a directory to hold the database
	path, _ := ioutil.TempDir("", "mdb_test")
	defer os.RemoveAll(path)

	// open the LMDB environment
	env, _ := lmdb.NewEnv()
	env.SetMaxDBs(1)
	env.Open(path, 0, 0664)
	defer env.Close()

	// open a database.
	txn, _ := env.BeginTxn(nil, 0)
	db, _ := txn.OpenDBI("exampledb", lmdb.Create)
	txn.Commit()

	// write some data
	txn, _ = env.BeginTxn(nil, 0)
	cursor, _ := txn.OpenCursor(db)
	cursor.Put([]byte("key0"), []byte("val0"), 0)
	cursor.Put([]byte("key1"), []byte("val1"), 0)
	cursor.Put([]byte("key2"), []byte("val2"), 0)
	cursor.Close()

	// inspect the transaction
	stat, _ := txn.Stat(db)
	fmt.Println(stat.Entries)

	// commit the transaction
	_ = txn.Commit()

	// scan the database
	txn, _ = env.BeginTxn(nil, lmdb.Readonly)
	defer txn.Abort()
	cursor, _ = txn.OpenCursor(db)
	defer cursor.Close()

	for {
		bkey, bval, err := cursor.Get(nil, nil, lmdb.Next)
		if err == lmdb.ErrNotFound {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s: %s\n", bkey, bval)
	}

	// Output:
	// 3
	// key0: val0
	// key1: val1
	// key2: val2
}