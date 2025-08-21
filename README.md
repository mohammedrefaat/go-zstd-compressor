# 🗜️ Zstd Compressor

A modern web-based file compression and decompression tool using the Zstandard (Zstd) algorithm. This portable application provides an intuitive interface for compressing files and folders into `.zst` archives and extracting them with automatic downloads.

## ✨ Features

- **🗂️ File Compression**: Compress multiple files and folders into high-efficiency Zstandard archives
- **📦 File Extraction**: Decompress `.zst` archives with automatic download of extracted content
- **⚙️ Adjustable Compression**: Choose from 19 different compression levels (1=Fastest to 19=Ultimate)
- **🎯 Drag & Drop Interface**: Intuitive UI with drag and drop support for files and folders
- **🔒 Cross-Platform Security**: Safe path handling for Windows, macOS, and Linux
- **🌐 Web-Based**: No installation required - runs entirely in your browser
- **⬇️ Auto-Downloads**: Seamless download experience in Chrome and other modern browsers
- **📊 Real-time Statistics**: View compression ratios, file sizes, and processing time

## 🚀 Quick Start

### Prerequisites
- Go 1.16 or later
- Modern web browser (Chrome, Firefox, Safari, Edge)

### Installation & Running

1. **Clone the repository:**
```bash
git clone https://github.com/your-username/zstd-compressor.git
cd zstd-compressor
```

2. **Install dependencies:**
```bash
go mod tidy
```

3. **Run the application:**
```bash
go run main.go
```

4. **Open your browser:**
   Navigate to `http://localhost:8080`

## 📖 How to Use

### 🗜️ Compressing Files

1. Open the application in your web browser
2. Stay on the **"Compress Files"** tab (default)
3. **Select files/folders:**
   - Click the file browser area, or
   - Drag and drop files/folders directly
   - Toggle between "Files" and "Folder" selection modes
4. **Adjust settings (optional):**
   - Set custom output filename
   - Choose compression level (1-19)
5. Click **"🗜️ Compress Files"**
6. **Automatic download** of the `.zst` archive will start
7. View compression statistics and download manually if needed

### 📂 Extracting Archives

1. Switch to the **"Decompress Archive"** tab
2. **Select archive:**
   - Click to browse for a `.zst` file, or
   - Drag and drop a `.zst` file
3. **Set extraction options (optional):**
   - Specify output directory name
4. Click **"📂 Extract Archive"**
5. **Automatic download** of extracted files as a ZIP will start
6. View extraction statistics

## 🔧 Building for Production

### Create Standalone Binary
```bash
# Build optimized binary with embedded frontend
go build -ldflags="-s -w" -o zstd-compressor

# Run the binary
./zstd-compressor
```

The resulting binary:
- ✅ Includes all frontend assets (no external files needed)
- ✅ Can be deployed to any system with matching architecture
- ✅ Portable - just copy and run

### Cross-Platform Builds
```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o zstd-compressor.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o zstd-compressor-mac

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o zstd-compressor-linux
```

## 🛠️ API Reference

The application provides RESTful API endpoints for programmatic access:

| Endpoint | Method | Description |
|----------|---------|-------------|
| `/api/compress` | POST | Compress uploaded files into `.zst` archive |
| `/api/decompress` | POST | Extract `.zst` archive |
| `/api/upload` | POST | Upload files for compression |
| `/api/upload-archive` | POST | Upload archive for extraction |
| `/api/download` | GET | Download compressed `.zst` file |
| `/api/download-extracted` | GET | Download extracted files as ZIP |
| `/api/list-files` | GET | List directory contents |

### Example API Usage

**Compress Files:**
```bash
# Upload files first
curl -X POST -F "files=@file1.txt" -F "files=@file2.txt" http://localhost:8080/api/upload

# Then compress (using returned file paths)
curl -X POST -H "Content-Type: application/json" \
  -d '{"files":["/tmp/file1.txt","/tmp/file2.txt"],"output":"my-archive","level":3}' \
  http://localhost:8080/api/compress
```

## 🏗️ Technical Architecture

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Backend** | Go with standard HTTP server | File processing and API endpoints |
| **Frontend** | HTML5, CSS3, Vanilla JavaScript | Modern responsive web interface |
| **Compression** | Zstandard via `klauspost/compress` | High-efficiency compression algorithm |
| **Archiving** | TAR format | Cross-platform file container |
| **Security** | Path sanitization & validation | Safe file handling across platforms |
| **Assets** | Go 1.16+ embed directive | Self-contained binary deployment |

## 🔐 Security Features

- **Path Traversal Protection**: Prevents `../` attacks during extraction
- **Input Sanitization**: Cleans file names and paths for all operating systems
- **Directory Containment**: Ensures extracted files stay within designated folders
- **Cross-Platform Safety**: Handles Windows drive letters and special characters
- **Memory Limits**: Configurable upload size limits to prevent abuse

## 🐛 Troubleshooting

### Common Issues

**"Directory syntax incorrect" on Windows:**
- ✅ Fixed in latest version with improved path sanitization

**Files not downloading automatically:**
- Ensure your browser allows automatic downloads
- Check if popup/download blocker is enabled
- Use the manual download buttons if needed

**Large file uploads failing:**
- Increase the upload size limit in `main.go` if needed
- Check available disk space in temp directory

## 📊 Compression Level Guide

| Level | Speed | Ratio | Best For |
|-------|-------|--------|----------|
| 1-3 | Fastest | Good | Quick daily backups |
| 4-6 | Fast | Better | General purpose |
| 7-12 | Medium | Very Good | Archival storage |
| 13-19 | Slower | Maximum | Long-term compression |

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and test thoroughly
4. Commit with clear messages: `git commit -m "Add feature description"`
5. Push and create a Pull Request

### Development Setup
```bash
# Clone your fork
git clone https://github.com/your-fork/zstd-compressor.git
cd zstd-compressor

# Install dependencies
go mod tidy

# Run in development mode
go run main.go

# Run tests
go test ./...
```

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🙏 Acknowledgments

- **Zstandard Algorithm**: Developed by Facebook for high-performance compression
- **klauspost/compress**: Excellent Go library providing Zstd implementation
- **Go Standard Library**: Robust HTTP server and file handling capabilities
- **Community**: Thanks to all contributors and users providing feedback

## 📞 Support

- 🐛 **Bug Reports**: [Open an issue](https://github.com/your-username/zstd-compressor/issues)
- 💡 **Feature Requests**: [Start a discussion](https://github.com/your-username/zstd-compressor/discussions)
- 📖 **Documentation**: Check the [Wiki](https://github.com/your-username/zstd-compressor/wiki)
- 💬 **Questions**: Use [GitHub Discussions](https://github.com/your-username/zstd-compressor/discussions)

---

⭐ **Star this repository if you find it useful!**

🔄 **Last Updated**: Features automatic downloads, Windows path fixes, and enhanced security