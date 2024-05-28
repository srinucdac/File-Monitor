package main

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestFileMonitoring(t *testing.T) {
	// Setup test environment
	testDir := "./test_watch"
	testDB := "./test_file_data.db"
	concurrency := 2

	os.Mkdir(testDir, 0755)
	defer os.RemoveAll(testDir)
	defer os.Remove(testDB)

	config = Config{
		Directory:   testDir,
		Database:    testDB,
		Concurrency: concurrency,
	}

	db, err := sql.Open("sqlite3", config.Database)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS file_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		byte_count INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("Error creating table: %v", err)
	}

	// Start file monitoring in a separate goroutine
	go main()

	// Create test files
	testFilePath := filepath.Join(testDir, "testfile.txt")
	err = os.WriteFile(testFilePath, []byte("Hi there!"), 0644)
	if err != nil {
		t.Fatalf("Error writing test file: %v", err)
	}

	// Allow some time for the file to be processed
	time.Sleep(2 * time.Second)

	// Verify data in the database
	var filePath string
	var byteCount int
	err = db.QueryRow(`SELECT file_path, byte_count FROM file_data WHERE file_path = ?`, testFilePath).Scan(&filePath, &byteCount)
	if err != nil {
		t.Fatalf("Error querying database: %v", err)
	}

	assert.Equal(t, testFilePath, filePath)
	assert.Equal(t, 8, byteCount) // "Hi there" is 8 bytes
}

func TestFileModification(t *testing.T) {
	// Setup test environment
	testDir := "./test_watch_mod"
	testDB := "./test_file_data_mod.db"
	concurrency := 2

	os.Mkdir(testDir, 0755)
	defer os.RemoveAll(testDir)
	defer os.Remove(testDB)

	config = Config{
		Directory:   testDir,
		Database:    testDB,
		Concurrency: concurrency,
	}

	db, err := sql.Open("sqlite3", config.Database)
	if err != nil {
		t.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS file_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		byte_count INTEGER NOT NULL
	)`)
	if err != nil {
		t.Fatalf("Error creating table: %v", err)
	}

	// Start file monitoring in a separate goroutine
	go main()

	// Create and modify test files
	testFilePath := filepath.Join(testDir, "testfile.txt")
	err = ioutil.WriteFile(testFilePath, []byte("Hello"), 0644)
	if err != nil {
		t.Fatalf("Error writing test file: %v", err)
	}

	time.Sleep(1 * time.Second) // Wait for the file to be processed

	err = ioutil.WriteFile(testFilePath, []byte("Hello, World! Updated!"), 0644)
	if err != nil {
		t.Fatalf("Error modifying test file: %v", err)
	}

	// Allow some time for the file to be processed
	time.Sleep(2 * time.Second)

	// Verify data in the database
	var filePath string
	var byteCount int
	err = db.QueryRow(`SELECT file_path, byte_count FROM file_data WHERE file_path = ? ORDER BY id DESC LIMIT 1`, testFilePath).Scan(&filePath, &byteCount)
	if err != nil {
		t.Fatalf("Error querying database: %v", err)
	}

	assert.Equal(t, testFilePath, filePath)
	assert.Equal(t, 23, byteCount) // "Hello, World! Updated!" is 23 bytes
}
