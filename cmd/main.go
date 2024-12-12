package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"mass-relay/internal/config"
	"mass-relay/internal/storage"
	"mass-relay/internal/ui"
	"mass-relay/internal/upload"
	"os"
	"path/filepath"
	"strconv"
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

	startTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxConcurrentUplaods)

	updateChan := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-updateChan:
				ui.UpdateDisplay(totalFiles, completedFiles, inProgressFiles, totalBytes, finishedBytes, errored, startTime)
			case <-ctx.Done():
				return
			}
		}
	}()

	updateChan <- struct{}{}

	for i, file := range files {
		sem <- struct{}{}
		wg.Add(1)
		go func(file string) {
			defer func() {
				<-sem

				fileInfo, _ := os.Stat(file)
				oppId := getId(prefixToID, fileInfo.Name())
				if oppId == "" {
					errored = append(errored, file)
				}
				inProgressFiles = removeFileFromInProgress(inProgressFiles, "Example file "+strconv.Itoa(i)+".pdf")
				completedFiles++
				finishedBytes += fileInfo.Size()
				updateChan <- struct{}{}
				wg.Done()
			}()

			fileInfo, _ := os.Stat(file)
			fileSize := fileInfo.Size()
			// oppId := getId(prefixToID, fileInfo.Name())
			inProgressFiles = append(inProgressFiles, "Example file "+strconv.Itoa(i)+".pdf")
			updateChan <- struct{}{}

			if isSimulation {
				time.Sleep(time.Duration(fileSize) * 5000)
			} else {
				err := upload.UploadFile(ctx, file, cfg.RemoteURL, cfg.Token)
				if err != nil {
					fmt.Printf("Error uploading %s: %v\n", file, err)
				}
			}
		}(file)
	}

	wg.Wait()
	close(updateChan)
	fmt.Println("Migration Complete!")
}
