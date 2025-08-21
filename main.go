package main

import (
	"archive/tar"
	"archive/zip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

//go:embed frontend/*
var embeddedFrontend embed.FS

type CompressRequest struct {
	Files  []string `json:"files"`
	Output string   `json:"output"`
	Level  int      `json:"level"`
}

type DecompressRequest struct {
	Archive   string `json:"archive"`
	OutputDir string `json:"outputDir"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type CompressionStats struct {
	OriginalSize     int64   `json:"originalSize"`
	CompressedSize   int64   `json:"compressedSize"`
	CompressionRatio float64 `json:"compressionRatio"`
	Duration         string  `json:"duration"`
	OutputFile       string  `json:"outputFile"`
}

type UploadResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		FilePaths []string `json:"filePaths,omitempty"`
		FilePath  string   `json:"filePath,omitempty"`
	} `json:"data,omitempty"`
}

func main() {
	// Serve embedded frontend files
	frontendFS, err := fs.Sub(embeddedFrontend, "frontend")
	if err != nil {
		log.Fatal("Failed to create frontend filesystem:", err)
	}

	http.Handle("/", http.FileServer(http.FS(frontendFS)))

	// API endpoints
	http.HandleFunc("/api/compress", handleCompress)
	http.HandleFunc("/api/decompress", handleDecompress)
	http.HandleFunc("/api/list-files", handleListFiles)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/upload-archive", handleUploadArchive)
	http.HandleFunc("/api/download", handleDownload)
	http.HandleFunc("/api/download-extracted", handleDownloadExtracted)

	port := "8080"
	fmt.Printf("Starting Zstd Compressor on http://localhost:%s\n", port)
	fmt.Println("Open your browser and navigate to the URL above")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleCompress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, false, "Invalid request format", nil)
		return
	}

	if len(req.Files) == 0 {
		sendResponse(w, false, "No files selected", nil)
		return
	}

	// Generate output filename if not provided
	if req.Output == "" {
		if len(req.Files) == 1 {
			baseName := filepath.Base(req.Files[0])
			if strings.Contains(baseName, ".") {
				baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
			}
			req.Output = baseName + ".zst"
		} else {
			req.Output = "archive.zst"
		}
	}

	if !strings.HasSuffix(req.Output, ".zst") {
		req.Output += ".zst"
	}

	// Set default compression level
	if req.Level < 1 || req.Level > 19 {
		req.Level = 3
	}

	stats, err := compressFiles(req.Files, req.Output, req.Level)
	if err != nil {
		sendResponse(w, false, fmt.Sprintf("Compression failed: %v", err), nil)
		return
	}

	sendResponse(w, true, "Compression completed successfully", stats)
}

func handleDecompress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DecompressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendResponse(w, false, "Invalid request format", nil)
		return
	}

	if req.Archive == "" {
		sendResponse(w, false, "No archive file specified", nil)
		return
	}

	// Generate output directory if not provided
	if req.OutputDir == "" {
		baseName := filepath.Base(req.Archive)
		if strings.HasSuffix(baseName, ".zst") {
			baseName = strings.TrimSuffix(baseName, ".zst")
		}
		req.OutputDir = sanitizeDirectoryName(baseName) + "_extracted"
	} else {
		req.OutputDir = sanitizeDirectoryName(req.OutputDir)
	}

	// Ensure we're using a simple directory name without path components
	req.OutputDir = filepath.Base(req.OutputDir)

	fileCount, outputPath, err := decompressFile(req.Archive, req.OutputDir)
	if err != nil {
		sendResponse(w, false, fmt.Sprintf("Decompression failed: %v", err), nil)
		return
	}

	data := map[string]interface{}{
		"extractedFiles": fileCount,
		"outputDir":      outputPath,
	}

	sendResponse(w, true, fmt.Sprintf("Decompression completed. Extracted %d files to %s", fileCount, req.OutputDir), data)
}

func compressFiles(files []string, outputFile string, level int) (*CompressionStats, error) {
	startTime := time.Now()

	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	// Create zstd encoder
	encoder, err := zstd.NewWriter(outFile, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(level)))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %v", err)
	}
	defer encoder.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(encoder)
	defer tarWriter.Close()

	var totalSize int64

	// Process each file
	for _, file := range files {
		if err := addToTar(tarWriter, file, &totalSize); err != nil {
			return nil, fmt.Errorf("failed to add %s to archive: %v", file, err)
		}
	}

	// Get final file stats
	stat, err := os.Stat(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get output file stats: %v", err)
	}

	stats := &CompressionStats{
		OriginalSize:     totalSize,
		CompressedSize:   stat.Size(),
		CompressionRatio: float64(stat.Size()) / float64(totalSize) * 100,
		Duration:         time.Since(startTime).String(),
		OutputFile:       outputFile,
	}

	return stats, nil
}

func addToTar(tarWriter *tar.Writer, filePath string, totalSize *int64) error {
	return filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Use relative path and sanitize it for cross-platform compatibility
		header.Name = path
		if filePath != path {
			relPath, err := filepath.Rel(filepath.Dir(filePath), path)
			if err != nil {
				return err
			}
			header.Name = filepath.Join(filepath.Base(filePath), relPath)
		}

		// Convert to forward slashes for tar format and sanitize
		header.Name = sanitizeTarPath(filepath.ToSlash(header.Name))

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}

			*totalSize += info.Size()
		}

		return nil
	})
}

func decompressFile(archiveFile, outputDir string) (int, string, error) {
	// Get the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get current directory: %v", err)
	}

	// Create the full output path in the current directory
	fullOutputDir := filepath.Join(cwd, outputDir)

	// Remove the directory if it already exists
	if _, err := os.Stat(fullOutputDir); err == nil {
		os.RemoveAll(fullOutputDir)
	}

	// Create the output directory
	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		return 0, "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Open archive file
	file, err := os.Open(archiveFile)
	if err != nil {
		return 0, "", fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	// Create zstd decoder
	decoder, err := zstd.NewReader(file)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create zstd decoder: %v", err)
	}
	defer decoder.Close()

	// Create tar reader
	tarReader := tar.NewReader(decoder)

	fileCount := 0

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, "", fmt.Errorf("failed to read tar header: %v", err)
		}

		// Sanitize the header name to prevent path traversal and invalid paths
		cleanName := sanitizeExtractPath(header.Name)
		if cleanName == "" {
			continue // Skip invalid paths
		}

		targetPath := filepath.Join(fullOutputDir, cleanName)

		// Ensure the target path is within the output directory (prevent path traversal)
		if !strings.HasPrefix(targetPath, filepath.Clean(fullOutputDir)+string(os.PathSeparator)) {
			continue // Skip paths that try to escape the output directory
		}

		// Ensure target directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return 0, "", fmt.Errorf("failed to create directory: %v", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return 0, "", fmt.Errorf("failed to create directory %s: %v", targetPath, err)
			}

		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return 0, "", fmt.Errorf("failed to create file %s: %v", targetPath, err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return 0, "", fmt.Errorf("failed to extract file %s: %v", targetPath, err)
			}

			fileCount++
		}
	}

	return fileCount, fullOutputDir, nil
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		sendUploadResponse(w, false, "Failed to parse form", nil)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		sendUploadResponse(w, false, "No files uploaded", nil)
		return
	}

	var filePaths []string

	// Create a temporary directory for uploaded files
	tempDir, err := ioutil.TempDir("", "zstd_upload")
	if err != nil {
		sendUploadResponse(w, false, "Failed to create temp directory", nil)
		return
	}

	// Save each file
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			sendUploadResponse(w, false, "Failed to open uploaded file", nil)
			return
		}
		defer file.Close()

		// Create destination file
		destPath := filepath.Join(tempDir, fileHeader.Filename)
		destFile, err := os.Create(destPath)
		if err != nil {
			sendUploadResponse(w, false, "Failed to create destination file", nil)
			return
		}
		defer destFile.Close()

		// Copy file content
		_, err = io.Copy(destFile, file)
		if err != nil {
			sendUploadResponse(w, false, "Failed to save file", nil)
			return
		}

		filePaths = append(filePaths, destPath)
	}

	data := map[string]interface{}{
		"filePaths": filePaths,
	}

	sendUploadResponse(w, true, "Files uploaded successfully", data)
}

func handleUploadArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		sendUploadResponse(w, false, "Failed to parse form", nil)
		return
	}

	file, fileHeader, err := r.FormFile("archive")
	if err != nil {
		sendUploadResponse(w, false, "Failed to get uploaded file", nil)
		return
	}
	defer file.Close()

	// Create a temporary directory for uploaded files
	tempDir, err := ioutil.TempDir("", "zstd_upload")
	if err != nil {
		sendUploadResponse(w, false, "Failed to create temp directory", nil)
		return
	}

	// Create destination file
	destPath := filepath.Join(tempDir, fileHeader.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		sendUploadResponse(w, false, "Failed to create destination file", nil)
		return
	}
	defer destFile.Close()

	// Copy file content
	_, err = io.Copy(destFile, file)
	if err != nil {
		sendUploadResponse(w, false, "Failed to save file", nil)
		return
	}

	data := map[string]interface{}{
		"filePath": destPath,
	}

	sendUploadResponse(w, true, "Archive uploaded successfully", data)
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("file")
	if filePath == "" {
		http.Error(w, "File parameter is required", http.StatusBadRequest)
		return
	}

	// Check if file exists and is accessible
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(filePath))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Serve the file
	http.ServeFile(w, r, filePath)
}

func handleDownloadExtracted(w http.ResponseWriter, r *http.Request) {
	dirPath := r.URL.Query().Get("dir")
	if dirPath == "" {
		http.Error(w, "Directory parameter is required", http.StatusBadRequest)
		return
	}

	// Check if directory exists and is accessible
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		http.Error(w, "Directory not found", http.StatusNotFound)
		return
	}

	// Create a zip file of the extracted directory
	zipPath := dirPath + ".zip"
	err := zipDirectory(dirPath, zipPath)
	if err != nil {
		http.Error(w, "Failed to create download package", http.StatusInternalServerError)
		return
	}
	defer os.Remove(zipPath) // Clean up after download

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(zipPath))
	w.Header().Set("Content-Type", "application/zip")

	// Serve the zip file
	http.ServeFile(w, r, zipPath)
}

func zipDirectory(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func listDirectory(dirPath string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []map[string]interface{}

	// Add parent directory entry if not root
	if dirPath != "/" && dirPath != "." {
		parent := filepath.Dir(dirPath)
		files = append(files, map[string]interface{}{
			"name":    "..",
			"path":    parent,
			"isDir":   true,
			"size":    0,
			"modTime": "",
		})
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		files = append(files, map[string]interface{}{
			"name":    entry.Name(),
			"path":    fullPath,
			"isDir":   entry.IsDir(),
			"size":    info.Size(),
			"modTime": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	return files, nil
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dirPath := r.URL.Query().Get("path")
	if dirPath == "" {
		dirPath, _ = os.Getwd()
	}

	files, err := listDirectory(dirPath)
	if err != nil {
		sendResponse(w, false, fmt.Sprintf("Failed to list directory: %v", err), nil)
		return
	}

	data := map[string]interface{}{
		"currentPath": dirPath,
		"files":       files,
	}

	sendResponse(w, true, "Directory listed successfully", data)
}

func sendResponse(w http.ResponseWriter, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Success: success,
		Message: message,
		Data:    data,
	}
	json.NewEncoder(w).Encode(response)
}

func sendUploadResponse(w http.ResponseWriter, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	response := UploadResponse{
		Success: success,
		Message: message,
	}

	if data != nil {
		if filePaths, ok := data.(map[string]interface{})["filePaths"]; ok {
			response.Data.FilePaths = filePaths.([]string)
		}
		if filePath, ok := data.(map[string]interface{})["filePath"]; ok {
			response.Data.FilePath = filePath.(string)
		}
	}

	json.NewEncoder(w).Encode(response)
}

// Sanitization functions to fix Windows path issues
func sanitizeDirectoryName(name string) string {
	// Remove any invalid characters for directory names
	invalidChars := regexp.MustCompile(`[<>:"|?*\\\/]`)
	name = invalidChars.ReplaceAllString(name, "_")

	// Remove any leading/trailing spaces and dots
	name = strings.Trim(name, " .")

	// Ensure the name is not empty
	if name == "" {
		name = "extracted"
	}

	// Limit length to prevent issues
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

func sanitizeTarPath(path string) string {
	// Remove drive letters and leading slashes/backslashes for cross-platform compatibility
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}

	// Remove leading slashes and backslashes
	path = strings.TrimLeft(path, "/\\")

	// Replace backslashes with forward slashes
	path = strings.ReplaceAll(path, "\\", "/")

	// Remove any remaining invalid characters
	invalidChars := regexp.MustCompile(`[<>:"|?*]`)
	path = invalidChars.ReplaceAllString(path, "_")

	return path
}

func sanitizeExtractPath(path string) string {
	// Remove drive letters and leading slashes/backslashes
	if len(path) >= 2 && path[1] == ':' {
		path = path[2:]
	}

	// Remove leading slashes and backslashes
	path = strings.TrimLeft(path, "/\\")

	// Skip empty paths or paths with only dots
	if path == "" || strings.Trim(path, ".") == "" {
		return ""
	}

	// Prevent path traversal
	if strings.Contains(path, "..") {
		return ""
	}

	// Replace forward slashes with OS-appropriate separators
	path = filepath.FromSlash(path)

	// Remove any remaining invalid characters for the current OS
	if filepath.Separator == '\\' { // Windows
		invalidChars := regexp.MustCompile(`[<>:"|?*]`)
		path = invalidChars.ReplaceAllString(path, "_")
	}

	return path
}
