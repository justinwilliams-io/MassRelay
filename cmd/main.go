package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"mass-relay/internal/config"
	"mass-relay/internal/model"
	"mass-relay/internal/storage"
	"mass-relay/internal/ui"
	"mass-relay/internal/upload"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func removeFileFromInProgress(files []string, fileToRemove string) []string {
	for i, f := range files {
		if f == fileToRemove {
			return append(files[:i], files[i+1:]...)
		}
	}
	return files
}

func getId(prefixToID map[string]string, filename string) string {
	returnId := ""
	for prefix, id := range prefixToID {
		hyphenatedPrefix := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(prefix, "|", "-"), "/", "-"), ":", "-"), "\"", "-")
		if strings.HasPrefix(filename, hyphenatedPrefix) {
			returnId = id
			break
		}
	}
	return returnId
}

func main() {
	var isSimulation bool
	flag.BoolVar(&isSimulation, "simulate", false, "Simulate file transfers")
	flag.Parse()

	startTime := time.Now()

	logFileName := fmt.Sprintf("error_log_%s.txt", startTime.Format("2006-01-02_15:04:05"))

	logfile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening log file:", err)
	}
	defer logfile.Close()

	csvFile, err := os.Open("ids.csv")
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	prefixToID := make(map[string]string)
	for _, record := range records {
		prefixToID[record[0]] = record[1]
	}

	var files []string
	if isSimulation {
		files, _ = storage.ScanFiles("./simulation-files")
	} else {
		files, err = storage.ScanFiles("./simulation-files")
		if err != nil {
			fmt.Println("Error scanning files:", err)
			return
		}
	}

	totalFiles := len(files)
	completedFiles := 0
	inProgressFiles := make([]string, 0, len(files))
	errored := make([]string, 0, len(files))

	configDir, _ := os.UserConfigDir()
	cfg, err := config.ReadConfig(filepath.Join(configDir, "mass-relay", "config.yaml"))
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	totalBytes := int64(0)
	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			fmt.Printf("Error getting file info for %s: %v\n", file, err)
			continue
		}
		totalBytes += fileInfo.Size()
	}

	finishedBytes := int64(0)

	ctx := context.WithValue(context.Background(), "logFile", logfile)

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxConcurrentUploads)

	messages := make(chan model.Message, 1)
	go func() {
		for {
			select {
			case msg := <-messages:
				switch msg.IsAdding {
				case true:
					inProgressFiles = append(inProgressFiles, msg.FileName)
				case false:
					inProgressFiles = removeFileFromInProgress(inProgressFiles, msg.FileName)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second / 9)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ui.UpdateDisplay(totalFiles, completedFiles, inProgressFiles, totalBytes, finishedBytes, errored, startTime, isSimulation)
			case <-ctx.Done():
				return
			}
		}
	}()

	var uploader model.Uploader = &upload.DefaultUploader{}

	for _, file := range files {
		wg.Add(1)
		sem <- struct{}{}
		go func(file string) {
			fileInfo, _ := os.Stat(file)
            messages <- model.Message{
                IsAdding: true,
                FileName: fileInfo.Name(),
            }

			queryParams := map[string]string{}

			if isSimulation {
				err := uploader.UploadFile(ctx, file, "http://localhost:8080", cfg.Token, queryParams)
				if err != nil {
					errored = append(errored, file)
					fmt.Printf("Error uploading %s: %v\n", file, err)
				}
			} else {
				err := uploader.UploadFile(ctx, file, cfg.RemoteURL, cfg.Token, queryParams)
				if err != nil {
					errored = append(errored, file)
					fmt.Printf("Error uploading %s: %v\n", file, err)
				}
			}

			completedFiles++
			finishedBytes += fileInfo.Size()

            messages <- model.Message{
                IsAdding: false,
                FileName: fileInfo.Name(),
            }
			wg.Done()

			<-sem
		}(file)
	}

	wg.Wait()
	ctx.Done()
	fmt.Println("Migration Complete!")
}
