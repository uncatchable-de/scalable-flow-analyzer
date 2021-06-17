package utils

import (
	"bufio"
	"fmt"
	gzip "github.com/klauspost/pgzip"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// CreateDir creates a new Directory
func CreateDir(dirname string) {
	err := os.MkdirAll(dirname, 0744)
	if err != nil {
		panic("Could not create export Directory: " + dirname + err.Error())
	}
}

// FileExists returns whether a File exists
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return true
}

// DirectoryExists returns whether a Directory exists
func DirectoryExists(directoryPath string) bool {
	info, err := os.Stat(directoryPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// GetPcapFiles returns all pcap files specified
func GetPcapFiles(input string) []string {
	switch {
	case FileExists(input):
		fmt.Println("Use input File:", input)
		return []string{input}
	case DirectoryExists(input):
		var files []string
		fmt.Println("Use input Directory:", input)
		err := filepath.Walk(input, func(filepath string, info os.FileInfo, err error) error {
			// Skip directories and subdirectories
			if info.IsDir() || path.Clean(path.Dir(filepath)) != path.Clean(input) {
				return nil
			}
			// Only append pcapng files
			if strings.HasSuffix(info.Name(), ".pcapng") || strings.HasSuffix(info.Name(), ".pcapng.bz2") {
				files = append(files, filepath)
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		sort.Slice(files, func(i, j int) bool { return files[i] < files[j] })
		fmt.Println("Analyze the following files in this order (please recheck!):")
		for _, file := range files {
			fmt.Println(file)
		}
		return files
	default:
		log.Fatal("Input does not exist. Please recheck", input)
		return []string{}
	}
}

// GetFilesInPath returns the complete path to all files in the inputDirectory which do have the required extension.
// Skips subdirectories
func GetFilesInPath(inputDirectory, extensionWithoutDot string) []string {
	if !DirectoryExists(inputDirectory) {
		return []string{}
	}
	var files []string
	err := filepath.Walk(inputDirectory, func(filepath string, info os.FileInfo, err error) error {
		// Skip directories and subdirectories
		if info.IsDir() || path.Clean(path.Dir(filepath)) != path.Clean(inputDirectory) {
			return nil
		}
		// Only append files with correct extension
		if strings.HasSuffix(info.Name(), "."+extensionWithoutDot) {
			files = append(files, filepath)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

// Returns the filename of a path.
func GetFilename(filePath string, withExtension bool) string {
	filename := filepath.Base(filePath)
	if withExtension {
		return filename
	}
	return filename[:strings.Index(filename, ".")]
}

// Returns whether the given file is a .bz2 zip file
func IsZipFile(filename string) (isZipFile bool) {
	if isBz2(filename) {
		return true
	}
	if isGzip(filename) {
		return true
	}
	return false
}

// Unzips a zip file and returns the filename of the unzipped file
func Unzip(filename string) (unzippedFileName string) {
	if isBz2(filename) {
		return unzipBz2(filename)
	}
	if isGzip(filename) {
		return unzipGzip(filename)
	}

	panic("File is neither a .b")
}

// Checks whether a file is a .bz2 file by looking at the extension
func isBz2(filename string) bool {
	return strings.HasSuffix(filename, ".bz2")
}

// Checks whether a file is a .gz file by looking at the extension
func isGzip(filename string) bool {
	return strings.HasSuffix(filename, ".gz")
}

// Unzips a .bz2 file and returns the name of the unzipped file
func unzipBz2(filename string) (unzippedFileName string) {
	var err error

	cmd := exec.Command("lbunzip2", "-k", "-n", "64", filename)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return filename[:len(filename)-4]
}

// Unzips a .gz file and returns the name of the unzipped file
func unzipGzip(filename string) (unzippedFileName string) {
	file, err := os.Open(filename)
	assertNoError(err)

	r, err := gzip.NewReader(file)
	assertNoError(err)

	unzippedFileName = filename[:len(filename)-3]
	f, err := os.Create(unzippedFileName)
	assertNoError(err)

	w := bufio.NewWriter(f)
	_, err = r.WriteTo(w)
	assertNoError(err)

	err = file.Close()
	assertNoError(err)

	return unzippedFileName
}

// Helper function that only returns if err == nil
func assertNoError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
