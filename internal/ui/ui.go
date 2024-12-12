package ui

import (
	"fmt"
	"strings"
	"time"
)

func UpdateDisplay(totalFiles int, completedFiles int, inProgressFiles []string, totalBytes, finishedBytes int64, errored []string, startTime time.Time) {
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
	fmt.Println("|")
	fmt.Println("--------------------------------------------------")

	progressBarLength := 50
	progressPercentage := float64(completedFiles) / float64(totalFiles)
	filledLength := int(progressPercentage * float64(progressBarLength))
	progressBar := strings.Repeat("=", filledLength) + strings.Repeat("-", progressBarLength-filledLength)
	fmt.Printf("\nProgress: [%s] %.2f%%\n", progressBar, progressPercentage*100)

	elapsedTime := time.Since(startTime)
	if elapsedTime > 0 && completedFiles > 0 {
		averageSpeed := float64(finishedBytes) / elapsedTime.Seconds()
		remainingTime := time.Duration(float64(totalBytes-finishedBytes)/averageSpeed) * time.Second
		fmt.Printf("Estimated Time Remaining: %s\n", remainingTime.String())
	}
}
