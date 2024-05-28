package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

type Config struct {
	Directory   string
	Database    string
	Concurrency int
}

var config Config

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config/")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}
}

func parseFlags() {
	dir := flag.String("directory", "", "Directory to watch")
	db := flag.String("database", "", "Path to SQLite database")
	concurrency := flag.Int("concurrency", 0, "Number of concurrent file processing goroutines")
	flag.Parse()

	if *dir != "" {
		config.Directory = *dir
	}
	if *db != "" {
		config.Database = *db
	}
	if *concurrency > 0 {
		config.Concurrency = *concurrency
	}
}

func main() {
	initConfig()
	parseFlags()

	db, err := sql.Open("sqlite3", config.Database)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS file_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT NOT NULL,
		byte_count INTEGER NOT NULL
	)`)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating watcher: %v", err)
	}
	defer watcher.Close()

	var wg sync.WaitGroup
	fileChan := make(chan string, config.Concurrency)

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go processFiles(fileChan, db, &wg)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
					fileChan <- event.Name
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Error: %v", err)
			}
		}
	}()

	err = watcher.Add(config.Directory)
	if err != nil {
		log.Fatalf("Error adding directory to watcher: %v", err)
	}

	wg.Wait()
}

func processFiles(fileChan chan string, db *sql.DB, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range fileChan {
		processFile(filePath, db)
	}
}

func processFile(filePath string, db *sql.DB) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error stating file: %v", err)
		return
	}

	if fileInfo.IsDir() {
		return
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("Error reading file: %v", err)
		return
	}

	byteCount := len(data)

	_, err = db.Exec(`INSERT INTO file_data (file_path, byte_count) VALUES (?, ?)`,
		filePath, byteCount)
	if err != nil {
		log.Printf("Error inserting data into database: %v", err)
	}
}
