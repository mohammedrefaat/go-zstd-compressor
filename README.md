# Zstd Compressor

A web-based file compression and decompression tool using the Zstandard (Zstd) algorithm. This application provides a user-friendly interface for compressing files and folders into `.zst` archives and extracting them back.

## Features

- **File Compression**: Compress multiple files and folders into Zstandard archives
- **File Extraction**: Decompress `.zst` archives back to their original form
- **Adjustable Compression**: Choose from 19 different compression levels (1-19)
- **Drag & Drop**: Intuitive interface with drag and drop support
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Web-Based**: No installation required - runs in your browser

## How to Use

### Compression
1. Open the application in your web browser
2. Select the "Compress Files" tab
3. Choose files or folders to compress (either by clicking or dragging)
4. Adjust compression level if desired (default is 3 - balanced)
5. Optionally specify an output filename
6. Click "Compress Files"
7. Download the resulting `.zst` archive

### Decompression
1. Select the "Decompress Archive" tab
2. Choose a `.zst` file to extract
3. Optionally specify an output directory name
4. Click "Extract Archive"
5. Download the extracted files as a ZIP archive

## Installation

### Prerequisites
- Go 1.16 or later
- Modern web browser

### Steps
1. Clone the repository:

git clone https://github.com/your-username/zstd-compressor.git
cd zstd-compressor

2. Install dependencies:
go mod download

3. Build and run the application:
go build -o zstd-compressor
./zstd-compressor

4. Open your browser and navigate to http://localhost:8080

API Endpoints
The application provides the following REST API endpoints:

POST /api/compress - Compress files

POST /api/decompress - Decompress archives

GET /api/list-files - List files in a directory

POST /api/upload - Upload files for compression

POST /api/upload-archive - Upload archives for decompression

GET /api/download - Download compressed files

GET /api/download-extracted - Download extracted files

Technical Details
Backend: Go with standard library HTTP server

Frontend: HTML5, CSS3, and vanilla JavaScript

Compression: Zstandard algorithm via klauspost/compress

Archiving: TAR format for storing multiple files

Embedded Assets: Frontend files embedded using Go 1.16+ embed directive

Building for Production
To create a standalone binary:
go build -ldflags="-s -w" -o zstd-compressor
The resulting binary includes all frontend assets and can be deployed to any system with the same architecture.

License
This project is open source and available under the MIT License.

Contributing
Contributions are welcome! Please feel free to submit a Pull Request.

Support
If you encounter any issues or have questions, please open an issue on the GitHub repository.

Acknowledgments
Zstandard compression algorithm by Facebook

klauspost/compress Go library for Zstandard support

Go standard library for HTTP server and file handling
