package main


import (
    "fmt"
    "io/ioutil"
    "encoding/xml"
    "os"
)

type Result struct {
    Files   []File   `xml:"file"`
}

type File struct {
    Name    string   `xml:"name,attr"`
    Errors  []Error  `xml:"error"`
}

type Error struct {
    Line     string   `xml:"line,attr"`
    Column   string   `xml:"column,attr"`
    Severity string   `xml:"severity,attr"`
    Message  string   `xml:"message,attr"`
    Source   string   `xml:"source,attr"`
}

func main() {
    xmlFile, err := os.Open("scheduler/target/checkstyle-result.xml")
    if err != nil {
        panic(err)
    }

    fmt.Println("Successfully Opened file")
    defer xmlFile.Close()

    byteValue, err := ioutil.ReadAll(xmlFile)
    if err != nil {
        panic(err)
    }

    var result Result
    err = xml.Unmarshal(byteValue, &result)
    if err != nil {
        panic(err)
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
      os.Exit(1)
    }
}
