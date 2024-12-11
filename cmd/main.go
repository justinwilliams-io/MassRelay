package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"mass-relay/internal/config"
	"mass-relay/internal/storage"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type FileInfo struct {
	Name string
	Size int64
}

func UploadFile(ctx context.Context, filePath string, url string, token string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, file)
	if err != nil {
		return fmt.Errorf("failed to create HTTP reqeust: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
	}

	return nil
}

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

func updateDisplay(totalFiles int, completedFiles int, inProgressFiles []string, totalBytes, finishedBytes int64, errored []string) {
	fmt.Print("\033[2J\033[H") // Clear the screen

	fmt.Println("--------------------------------------------------")
	fmt.Println("| File Migration Tool |")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("| Total Files: %d | Completed: %d/%d\n", totalFiles, completedFiles, totalFiles)
	fmt.Println("|")
	fmt.Println("| In Progress:")
	for _, file := range inProgressFiles {
		fmt.Printf("|   - %s\n", file)
	}
	fmt.Println("|")
	fmt.Println("| Errored:") // Replace 0 with the actual number of errors
	for _, file := range errored {
		fmt.Printf("|   - %s\n", file)
	}
	fmt.Println("--------------------------------------------------")

	progressBarLength := 50
	progressPercentage := float64(completedFiles) / float64(totalFiles)
	filledLength := int(progressPercentage * float64(progressBarLength))
	progressBar := strings.Repeat("=", filledLength) + strings.Repeat("-", progressBarLength-filledLength)
	fmt.Printf("\nProgress: [%s] %.2f%%\n", progressBar, progressPercentage*100)

	remainingTime := time.Duration(totalBytes-finishedBytes) * 5000
	fmt.Printf("Estimated Time Remaining: %s\n", remainingTime.String())
}

func main() {
	isSimulation := true

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxConcurrentUplaods)

	updateChan := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-updateChan:
				updateDisplay(totalFiles, completedFiles, inProgressFiles, totalBytes, finishedBytes, errored)
			case <-ctx.Done():
				return
			}
		}
	}()

	updateChan <- struct{}{}

	for _, file := range files {
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
				inProgressFiles = removeFileFromInProgress(inProgressFiles, fileInfo.Name()+" - "+oppId)
				completedFiles++
				finishedBytes += fileInfo.Size()
				updateChan <- struct{}{}
				wg.Done()
			}()

			fileInfo, _ := os.Stat(file)
			fileSize := fileInfo.Size()
			oppId := getId(prefixToID, fileInfo.Name())
			inProgressFiles = append(inProgressFiles, fileInfo.Name()+" - "+oppId)
			updateChan <- struct{}{}

			if isSimulation {
				time.Sleep(time.Duration(fileSize) * 5000)
			} else {
				err := UploadFile(ctx, file, cfg.RemoteURL, cfg.Token)
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
