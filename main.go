package main

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "sync"
)

type MatchResult struct {
    Filename    string
    LineNumber  int
    MatchedLine string
    Error       error
}

func CheckFile(filename, pattern string, ch chan<- MatchResult, wg *sync.WaitGroup) {
    defer wg.Done()
    fmt.Println("Checking file: ", filename)
    
    regex, err := regexp.Compile(pattern)
    if err != nil {
        ch <- MatchResult{Filename: filename, Error: fmt.Errorf("invalid pattern: %v", err)}
        return
    }

    file, err := os.Open(filename)
    if err != nil {
        ch <- MatchResult{Filename: filename, Error: fmt.Errorf("error opening file: %v", err)}
        return
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    lineNum := 0
    for scanner.Scan() {
        lineNum++
        line := scanner.Text()
        if regex.MatchString(line) {
            ch <- MatchResult{
                Filename:    filename,
                LineNumber:  lineNum,
                MatchedLine: line,
            }
        }
    }

    if err := scanner.Err(); err != nil {
        ch <- MatchResult{Filename: filename, Error: fmt.Errorf("error reading file: %v", err)}
    }
}

func SearchDirectory(dir, pattern string) {
    resultChan := make(chan MatchResult)
    var wg sync.WaitGroup
    fileCount := 0

    // First pass: count files and add to waitgroup
    err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Ext(path) == ".txt" {
            fileCount++
        }
        return nil
    })

    if err != nil {
        fmt.Printf("Error walking directory: %v\n", err)
        return
    }

    fmt.Printf("Found %d files to process\n", fileCount)

    // Second pass: start goroutines
    err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && filepath.Ext(path) == ".txt" {
            fmt.Println("Processing file: ", path)
            wg.Add(1)
            go CheckFile(path, pattern, resultChan, &wg)
        }
        return nil
    })

    if err != nil {
        fmt.Printf("Error walking directory: %v\n", err)
        return
    }

    // Create a done channel to signal when all results are processed
    done := make(chan bool)

    // Start a goroutine to process results
    go func() {
        for result := range resultChan {
            if result.Error != nil {
                fmt.Printf("Error in file %s: %v\n", result.Filename, result.Error)
                continue
            }
            if result.MatchedLine != "" {
                relPath, err := filepath.Rel(".", result.Filename)
                if err != nil {
                    relPath = result.Filename
                }
                fmt.Printf("\nMatch found in: %s\nLine %d: %s\n",
                    relPath, result.LineNumber, result.MatchedLine)
            }
        }
        done <- true
    }()

    // Wait for all file processing goroutines to complete
    wg.Wait()
    
    // Close the result channel after all files are processed
    close(resultChan)
    
    // Wait for all results to be processed
    <-done
}

func main() {
    if len(os.Args) != 3 {
        fmt.Println("Usage: program <directory> <pattern>")
        return
    }

    directory := os.Args[1]
    pattern := os.Args[2]

    fmt.Printf("Searching for pattern '%s' in directory '%s'...\n\n", pattern, directory)
    SearchDirectory(directory, pattern)
    fmt.Println("Search complete!")
}
