package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DownloadFile downloads any file (video, app, image, etc.) and saves it locally.
func DownloadFile(urlStr string) (string, string, error) {
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	log.Println("Downloading:", urlStr)

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	//setFirefoxHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Handle gzip encoding if needed
	var body io.ReadCloser = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		body, err = gzip.NewReader(resp.Body)
		if err != nil {
			return "", "", fmt.Errorf("gzip error: %w", err)
		}
		defer body.Close()
	}

	// Determine filename and MIME type
	mimeType := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	filename, err := determineFilename(urlStr, resp.Header)
	if err != nil {
		return "", "", fmt.Errorf("failed to determine filename: %w", err)
	}
	if filepath.Ext(filename) == "" {
		if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
			filename += exts[0]
		}
	}
	filename = sanitizeFilename(filename)
	filePath := filepath.Join(cacheDir, filename)

	// Save the file
	out, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	start := time.Now()
	written, err := io.Copy(out, body)
	if err != nil {
		_ = os.Remove(filePath)
		return "", "", fmt.Errorf("failed to write file: %w", err)
	}
	duration := time.Since(start)
	speedKB := float64(written) / duration.Seconds() / 1024

	log.Printf("✅ Downloaded: %s (%.2f MB in %.2fs at %.2f KB/s)",
		filename, float64(written)/1024/1024, duration.Seconds(), speedKB)

	return filePath, mimeType, nil
}

func sanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := re.ReplaceAllString(filename, "_")
	safe = strings.Trim(safe, " .")
	if safe == "" {
		safe = "file_" + time.Now().Format("20060102_150405")
	}
	return safe
}

func setFirefoxHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
}

func determineFilename(urlStr string, headers http.Header) (string, error) {
	// 1. First try Content-Disposition header
	if cd := headers.Get("Content-Disposition"); cd != "" {
		if filename := parseContentDisposition(cd); filename != "" {
			return filename, nil
		}
	}

	// 2. Try to extract from URL path
	if u, err := url.Parse(urlStr); err == nil {
		if path := u.Path; path != "" && path != "/" {
			if name := filepath.Base(path); name != "" && name != "." && name != "/" {
				return name, nil
			}
		}
	}

	// 3. Fallback to URL-based name
	return generateUrlBasedFilename(urlStr), nil
}

func generateUrlBasedFilename(urlStr string) string {
	// Clean the URL to create a filename
	clean := strings.NewReplacer(
		"https://", "",
		"http://", "",
		"www.", "",
		"/", "_",
		"?", "_",
		"&", "_",
		"=", "_",
	).Replace(urlStr)

	// Limit length and add timestamp
	if len(clean) > 50 {
		clean = clean[:50]
	}
	return "download_" + time.Now().Format("20060102_150405") + "_" + clean
}

func parseContentDisposition(cd string) string {
	// Look for filename="..." or filename=...
	re := regexp.MustCompile(`filename\*?=['"]?(?:UTF-\d['"]*)?([^;"']*)['"]?`)
	matches := re.FindStringSubmatch(cd)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
