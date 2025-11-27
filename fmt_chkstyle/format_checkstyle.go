package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type Result struct {
	Files []File `xml:"file"`
}

type File struct {
	Name           string          `xml:"name,attr"`
	Errors         []CheckstyleErr `xml:"error"`
}

type CheckstyleErr struct {
	Line     string `xml:"line,attr"`
	Column   string `xml:"column,attr"`
	Severity string `xml:"severity,attr"`
	Message  string `xml:"message,attr"`
	Source   string `xml:"source,attr"`
}

func run() error {
	xmlFile, err := os.Open("scheduler/target/checkstyle-result.xml")
	if err != nil {
		return fmt.Errorf("failed to open checkstyle result file: %w", err)
	}

	fmt.Println("Successfully Opened file")
	defer xmlFile.Close()

	byteValue, err := io.ReadAll(xmlFile)
	if err != nil {
		return fmt.Errorf("failed to read checkstyle result file: %w", err)
	}

	var result Result
	err = xml.Unmarshal(byteValue, &result)
	if err != nil {
		return fmt.Errorf("failed to unmarshal XML: %w", err)
	}

	count := 0
	for _, f := range result.Files {
		for _, e := range f.Errors {
			count++
			fmt.Printf("::%s file=%s,line=%s,col=%s::%s\n", e.Severity, f.Name, e.Line, e.Column, e.Message)
		}
	}

	fmt.Printf("Total %d Issues\n", count)
	if count > 0 {
		return fmt.Errorf("found %d checkstyle issues", count)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
