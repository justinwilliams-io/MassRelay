package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func retry(fn func() error, filePath string, maxTries int, initialRetryDelay time.Duration, ctx context.Context) error {
    logFile := ctx.Value("logFile").(*os.File)
    log.SetOutput(logFile)
	for i := 0; i <= maxTries; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		if i == maxTries {
            log.Printf("Error uploading %s: %v", filePath, err)
			return fmt.Errorf("Failed to Upload File: %w", err)
		}
		time.Sleep(initialRetryDelay)
		initialRetryDelay *= 2
	}
	return fmt.Errorf("Failed after max attempts: %d", maxTries)
}

type DefaultUploader struct{}

func (u *DefaultUploader) UploadFile(ctx context.Context, filePath string, baseUrl string, token string, queryParams map[string]string) error {
	return retry(func() error {
        u, err := url.Parse(baseUrl)
        if err != nil {
            return fmt.Errorf("Failed to parse URL: %w", err)
        }

        q := u.Query()
        for key, value := range queryParams {
            q.Add(key, value)
        }

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		part, err := writer.CreateFormFile("Files", filepath.Base(filePath))

		_, err = io.Copy(part, file)
		if err != nil {
			return fmt.Errorf("failed to copy file to multipart form: %w", err)
		}

		err = writer.Close()
		if err != nil {
			return fmt.Errorf("failed to close multipart writer: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", u.String(), body)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send HTTP request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected HTTP status code: %d", resp.StatusCode)
		}

		// Handle the response body (e.g., parse JSON, check for errors)
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Parse the response JSON (if applicable)
		var responseBody map[string]interface{}
		err = json.Unmarshal(bodyBytes, &responseBody)
		if err != nil {
			return fmt.Errorf("failed to parse response JSON: %w", err)
		}

		// Check for errors in the response
		errors, ok := responseBody["Errors"].([]interface{})
		if !ok || len(errors) > 0 {
			return fmt.Errorf("Upload failed: %v", errors)
		}

        return nil
	}, filePath, 3, time.Second, ctx)
}
