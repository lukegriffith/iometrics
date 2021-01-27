// sqllite example used
// https://www.codeproject.com/Articles/5261771/Golang-SQLite-Simple-Example
// counter example used
// https://play.golang.org/p/1eDuXo2BkoO
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

func main() {

	recoverPtr := flag.Bool("recovery", true, "remove database file before run.")
	cleanPtr := flag.Bool("cleanup", true, "cleanup database between insert sample")
	insertWaitPtr := flag.Int("insertWait", 10, "wait time between inserts in millisecond.")
	sleepWaitPtr := flag.Int("sleepWait", 10, "wait time for inserts to complete in seconds.")
	insertCountPtr := flag.Int("insertCount", 10000, "how many uuids to insert into the SqlLite database.")
	metricsPortPtr := flag.String("metricsPort", ":8080", "TCP port for metrics server to run from.")
	flag.Parse()

	if *recoverPtr {
		err := os.Remove("sqlite-database.db") // I delete the file to avoid duplicated records.
		if err != nil {
			log.Println("no database file present.")
		}
	}

	insertWait := time.Duration(*insertWaitPtr)
	sleepWait := time.Duration(*sleepWaitPtr)
	insertCount := *insertCountPtr
	metricsPort := *metricsPortPtr
	// SQLite is a file based database.

	var passC count32
	var failC count32
	test := &test{
		&passC,
		&failC,
		false,
		0,
		make(map[string]int32),
		0,
	}

	// configure metrics server
	metrics := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, test.PrometheusMetric(insertCount, *insertWaitPtr, *sleepWaitPtr))
	}
	http.HandleFunc("/metrics", metrics)
	go http.ListenAndServe(metricsPort, nil)

	// start io testing.
	for true {

		if *cleanPtr {
			err := os.Remove("sqlite-database.db") // I delete the file to avoid duplicated records.
			if err != nil {
				log.Println("no database file present.")
			}
		}
		// sqllite creation
		log.Println("Creating sqlite-database.db...")
		file, err := os.Create("sqlite-database.db") // Create SQLite file
		if err != nil {
			log.Fatal(err.Error())
		}
		file.Close()
		log.Println("sqlite-database.db created")

		sqliteDatabase, err := sql.Open("sqlite3", "./sqlite-database.db") // Open the created SQLite File

		defer sqliteDatabase.Close() // Defer Closing the database

		//defer test.Reset()

		createTable(sqliteDatabase) // Create Database Tables

		insertTestData := `INSERT INTO testdata(text1, text1) VALUES (?, ?)`
		statement, err := sqliteDatabase.Prepare(insertTestData) // Prepare statement.

		// This is good to avoid SQL injections
		if err != nil {
			log.Fatalln(err.Error())
		}
		// INSERT RECORDS
		log.Println("Starting inserts.")
		for i := 1; i < insertCount; i++ {
			go insertTest(sqliteDatabase, statement, test)
			time.Sleep(insertWait * time.Millisecond)
		}
		log.Println("Starating sleep.")
		time.Sleep(sleepWait * time.Second)

		log.Println("Display")
		// DISPLAY INSERTED RECORDS
		displayTest(sqliteDatabase, 10)

		dropTable(sqliteDatabase)

		test.Report()

		// to test to stdout. using prometheus npw.
		//test.Report()

	}

}

func createTable(db *sql.DB) {
	createStudentTableSQL := `CREATE TABLE testdata (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"text1" 		TEXT,
		"text2" 		TEXT
	  );` // SQL Statement for Create Table

	log.Println("Create test table...")

	statement, err := db.Prepare(createStudentTableSQL) // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec() // Execute SQL Statements
	log.Println("test table created")
}

func dropTable(db *sql.DB) {
	createStudentTableSQL := "drop TABLE testdata" // SQL Statement for Create Table

	log.Println("drop test table...")

	statement, err := db.Prepare(createStudentTableSQL) // Prepare SQL Statement
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec() // Execute SQL Statements
	log.Println("test table dropped.")
}

// We are passing db reference connection from main to our method with other parameters
func insertTest(db *sql.DB, statement *sql.Stmt, test *test) {
	text1 := uuid.NewString()
	text2 := uuid.NewString()
	_, err := statement.Exec(text1, text2)
	if err != nil {
		test.Fail(err.Error())
		return
	}
	test.Pass()
}

func displayTest(db *sql.DB, limit int) {

	row, err := db.Query(fmt.Sprintf("SELECT * FROM testdata LIMIT %d", limit))
	if err != nil {
		log.Fatal(err)
	}
	defer row.Close()
	for row.Next() { // Iterate and fetch the records from result cursor
		var id int
		var text1 string
		var text2 string
		row.Scan(&id, &text1, &text2)
		log.Println("test: ", id, " ", text1, " ", text2)
	}
}

type count32 int32

func (c *count32) Inc() int32 {
	return atomic.AddInt32((*int32)(c), 1)
}

func (c *count32) Get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

type test struct {
	pass             *count32
	fail             *count32
	failure          bool
	firstFailureSeen int32
	failureMessages  map[string]int32
	resetCount       int32
}

func (t *test) Pass() {
	t.pass.Inc()
}

func (t *test) Fail(err string) {
	if val, ok := t.failureMessages[err]; ok {
		t.failureMessages[err] = val + 1
	} else {
		t.failureMessages[err] = 1
	}

	t.fail.Inc()
}

func (t *test) Reset() {
	var passC count32
	var failC count32
	t.pass = &passC
	t.fail = &failC
	t.failureMessages = make(map[string]int32)
	t.resetCount++
}

func (t *test) Report() {
	log.Println("pass", t.pass.Get())
	log.Println("fail", t.fail.Get())
	log.Println("first failure", t.firstFailureSeen)
	log.Println("failure reasons", t.failureMessages)
}

func (t *test) PrometheusMetric(insertCount int, insertWait int, sleepWait int) string {
	pass := fmt.Sprintf("iometrics_sqllite_pass{insertCount=\"%d\",insertWait=\"%d\",sleepWait=\"%d\"} %d", insertCount, insertWait, sleepWait, t.pass.Get())
	fail := fmt.Sprintf("iometrics_sqllite_fail{insertCount=\"%d\",insertWait=\"%d\",sleepWait=\"%d\"} %d", insertCount, insertWait, sleepWait, t.fail.Get())
	//firstFailure := fmt.Sprintf("iometrics_sqllite_firstFailure{insertCount=\"%d\",insertWait=\"%d\",sleepWait=\"%d\"} %d", insertCount, insertWait, sleepWait, t.firstFailureSeen)

	metric := fmt.Sprintf(`
%s
%s`, pass, fail)

	return metric
}
