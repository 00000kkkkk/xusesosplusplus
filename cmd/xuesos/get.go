package xuesos

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func runGet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("get requires a package path\nUsage: xuesos get <github.com/user/repo>")
	}

	pkg := args[0]

	// Parse package path
	parts := strings.Split(pkg, "/")
	if len(parts) < 3 {
		return fmt.Errorf("invalid package path %q\nExpected: github.com/user/repo", pkg)
	}

	host := parts[0]
	user := parts[1]
	repo := parts[2]

	if host != "github.com" {
		return fmt.Errorf("only github.com packages are supported")
	}

	fmt.Printf("xuesos get: fetching %s...\n", pkg)

	// Create modules directory
	modulesDir := filepath.Join("xpp_modules", user, repo)
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return fmt.Errorf("cannot create modules directory: %w", err)
	}

	// Try to download main.xpp and any .xpp files from the repo root
	// Use GitHub raw content URL
	baseURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main", user, repo)

	// Try common file names
	files := []string{"main.xpp", "lib.xpp", "index.xpp", repo + ".xpp"}
	downloaded := 0

	for _, file := range files {
		url := baseURL + "/" + file
		resp, err := http.Get(url)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}
		if err != nil {
			continue
		}

		outPath := filepath.Join(modulesDir, file)
		if err := os.WriteFile(outPath, body, 0644); err != nil {
			return fmt.Errorf("cannot write %s: %w", outPath, err)
		}

		fmt.Printf("  downloaded: %s (%d bytes)\n", file, len(body))
		downloaded++
	}

	// Also try to get a file listing from the GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents", user, repo)
	resp, err := http.Get(apiURL)
	if err == nil && resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Simple JSON parsing - look for .xpp filenames
		content := string(body)
		// Find "name":"*.xpp" patterns
		for {
			idx := strings.Index(content, `"name":"`)
			if idx == -1 {
				break
			}
			content = content[idx+8:]
			endIdx := strings.Index(content, `"`)
			if endIdx == -1 {
				break
			}
			name := content[:endIdx]
			content = content[endIdx:]

			if strings.HasSuffix(name, ".xpp") {
				// Download this file
				fileURL := baseURL + "/" + name
				fileResp, fileErr := http.Get(fileURL)
				if fileErr != nil {
					continue
				}
				fileBody, fileErr := io.ReadAll(fileResp.Body)
				fileResp.Body.Close()
				if fileErr != nil {
					continue
				}

				outPath := filepath.Join(modulesDir, name)
				if _, statErr := os.Stat(outPath); statErr != nil {
					// File doesn't exist yet
					if writeErr := os.WriteFile(outPath, fileBody, 0644); writeErr == nil {
						fmt.Printf("  downloaded: %s (%d bytes)\n", name, len(fileBody))
						downloaded++
					}
				}
			}
		}
	} else if resp != nil {
		resp.Body.Close()
	}

	if downloaded == 0 {
		// Clean up empty directory
		os.RemoveAll(filepath.Join("xpp_modules", user, repo))
		return fmt.Errorf("no .xpp files found in %s", pkg)
	}

	// Write lock file
	lockPath := "xuesos.lock"
	lockEntry := fmt.Sprintf("%s\n", pkg)

	// Append to lock file (or create)
	f, err := os.OpenFile(lockPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot write lock file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(lockEntry); err != nil {
		return fmt.Errorf("cannot write lock entry: %w", err)
	}

	fmt.Printf("xuesos get: installed %s (%d files)\n", pkg, downloaded)
	return nil
}
