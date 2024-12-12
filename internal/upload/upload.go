package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func UploadFile(ctx context.Context, filePath string, url string, token string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("file", filepath.Base(filePath))
    if err != nil {
        return fmt.Errorf("failed to create form file: %w", err)
    }


    _, err = io.Copy(part, file)
    if err != nil {
        return fmt.Errorf("failed to copy file to multipart form: %w", err)
    }

    err = writer.Close()
    if err != nil {
        return fmt.Errorf("failed to close multipart writer: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, "POST", url, body)
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
    var responseBody interface{}
    err = json.Unmarshal(bodyBytes, &responseBody)
    if err != nil {
        return fmt.Errorf("failed to parse response JSON: %w", err)
    }

    // Check for errors in the response
    if err := checkResponseForErrors(responseBody); err != nil {

        return err
    }

    // Handle the successful response (e.g., update UI, create records)
    // ...


    return nil
}
