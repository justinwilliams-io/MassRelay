package main

import (
	"context"
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

func updateDisplay(totalFiles int, completedFiles int, inProgressFiles []string) {
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
	// Add estimated time remaining here, if applicable
	// fmt.Println("| Estimated Time Remaining: 15 minutes")
	fmt.Println("| Errors: 0") // Replace 0 with the actual number of errors
	fmt.Println("--------------------------------------------------")

	progressBarLength := 50
	progressPercentage := float64(completedFiles) / float64(totalFiles)
	filledLength := int(progressPercentage * float64(progressBarLength))
	progressBar := strings.Repeat("=", filledLength) + strings.Repeat("-", progressBarLength-filledLength)
	fmt.Printf("\nProgress: [%s] %.2f%%\n", progressBar, progressPercentage*100)
}

func main() {
	isSimulation := true

	var files []string
	var err error
	if isSimulation {
		files, _ = storage.ScanFiles("./simulation-files")
	} else {
		files, err = storage.ScanFiles(".")
		if err != nil {
			fmt.Println("Error scanning files:", err)
			return
		}
	}

	totalFiles := len(files)
	completedFiles := 0
	inProgressFiles := make([]string, 0, len(files))

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxConcurrentUplaods)

	updateChan := make(chan struct{}, 1)
    go func() {
        for {
            select {
            case <-updateChan:
                updateDisplay(totalFiles, completedFiles, inProgressFiles)
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
				wg.Done()

				completedFiles++
				inProgressFiles = removeFileFromInProgress(inProgressFiles, file)
				updateChan <- struct{}{}
			}()

			inProgressFiles = append(inProgressFiles, file)
			updateChan <- struct{}{}

			if isSimulation {
				fileInfo, _ := os.Stat(file)
				fileSize := fileInfo.Size()
				time.Sleep(time.Duration(fileSize) * 10000)
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
	fmt.Println("All files uploaded successfully!")
}
