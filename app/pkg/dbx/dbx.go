package dbx

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/WeCanHearYou/wechy/app/pkg/env"

	//required
	_ "github.com/lib/pq"
)

// New creates a new Database instance
func New() (*Database, error) {
	conn, err := sql.Open("postgres", env.MustGet("DATABASE_URL"))
	if err != nil {
		return nil, throw(err)
	}

	db := &Database{conn}
	if env.IsTest() {
		db.load("/app/pkg/dbx/setup.sql")
	}
	return db, nil
}

// Database represents a connection to a SQL database
type Database struct {
	conn *sql.DB
}

// Execute given SQL command
func (db Database) Execute(command string, args ...interface{}) error {
	_, err := db.conn.Exec(command, args...)
	if err != nil {
		return throw(err)
	}
	return nil
}

// QueryInt executes given SQL command and return first integer of first row
func (db Database) QueryInt(command string, args ...interface{}) (int, error) {
	var data int
	row := db.conn.QueryRow(command, args...)
	err := row.Scan(&data)
	if err != nil {
		return 0, err
	}
	return data, nil
}

// QueryString executes given SQL command and return first string of first row
func (db Database) QueryString(command string, args ...interface{}) (string, error) {
	var data string
	row := db.conn.QueryRow(command, args...)
	err := row.Scan(&data)
	if err != nil {
		return "", err
	}
	return data, nil
}

// Query the database with given SQL command
func (db Database) Query(command string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(command, args...)
}

// QueryRow the database with given SQL command and returns only one row
func (db Database) QueryRow(command string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(command, args...)
}

// Exists returns true if at least one record is found
func (db Database) Exists(command string, args ...interface{}) (bool, error) {
	rows, err := db.conn.Query(command, args...)
	if rows != nil {
		defer rows.Close()
		return rows.Next(), nil
	}
	return false, err
}

// Count returns number of rows
func (db Database) Count(command string, args ...interface{}) (int, error) {
	rows, err := db.conn.Query(command, args...)
	defer rows.Close()
	count := 0
	for rows != nil && rows.Next() {
		count++
	}
	return count, err
}

// Begin returns a new SQL transaction
func (db Database) Begin() (*sql.Tx, error) {
	return db.conn.Begin()
}

// Get first row and bind to given data
func (db Database) Get(data interface{}, command string, args ...interface{}) error {
	rows, err := db.Query(command, args...)
	if err != nil {
		return err
	}

	if rows.Next() {
		return scan(rows, data)
	}
	return nil
}

//Select all matched rows bind to given data
func (db Database) Select(data interface{}, command string, args ...interface{}) error {
	rows, err := db.Query(command, args...)
	if err != nil {
		return err
	}

	sliceType := reflect.TypeOf(data).Elem()
	items := reflect.New(sliceType).Elem()
	for rows.Next() {
		item := reflect.New(sliceType.Elem())
		err = scan(rows, item.Interface())
		if err != nil {
			return err
		}
		items = reflect.Append(items, item.Elem())
	}

	reflect.Indirect(reflect.ValueOf(data)).Set(items)
	return nil
}

func scan(rows *sql.Rows, data interface{}) error {
	columns, _ := rows.Columns()
	fields := make(map[string]interface{})

	s := reflect.ValueOf(data).Elem()
	for i := 0; i < s.NumField(); i++ {
		field := s.Type().Field(i)
		tag := field.Tag.Get("db")
		if field.Type.Kind() != reflect.Ptr {
			if tag != "" {
				fields[tag] = s.Field(i).Addr().Interface()
			}
		} else {
			obj := reflect.New(s.Field(i).Type().Elem()).Elem()
			s.Field(i).Set(obj.Addr())
			nested := s.Field(i).Elem()
			for j := 0; j < nested.NumField(); j++ {
				nestedField := nested.Type().Field(j)
				nestedTag := nestedField.Tag.Get("db")
				if nestedTag != "" {
					fmt.Println(tag + "_" + nestedTag)
					fields[tag+"_"+nestedTag] = nested.Field(j).Addr().Interface()
				}
			}
		}
	}

	pointers := make([]interface{}, len(columns))
	for i, column := range columns {
		if pointer, ok := fields[column]; ok {
			pointers[i] = pointer
		} else {
			panic(fmt.Sprintf("No target for column %s", column))
		}
	}

	err := rows.Scan(pointers...)
	if err != nil {
		return err
	}
	return nil
}

// Close connection to database
func (db Database) Close() {
	if db.conn != nil {
		throw(db.conn.Close())
	}
}

func (db Database) load(path string) {
	content, err := ioutil.ReadFile(env.Path(path))
	if err != nil {
		panic(err)
	}

	err = db.Execute(string(content))
	if err != nil {
		panic(err)
	}
}

func throw(err error) error {
	if err != nil && env.IsTest() {
		panic(err)
	}
	return err
}