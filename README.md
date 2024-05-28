Overview:
This project provides a solution for monitoring file events within a specified directory and handling these events with concurrency control. 
It uses a goroutine-based event executor to efficiently manage file operations like creation, modification, and deletion.

Build Project : 
Create a config/config.yaml file with your specific settings for the directory to watch, the database path, and the concurrency level

Compile and run the Go application using:
  go run main.go

Optionally, you can override configuration settings using command-line arguments:
  go run main.go --directory "C:/new/path/to/watch" --database "C:/new/path/to/database.db" --concurrency 10

