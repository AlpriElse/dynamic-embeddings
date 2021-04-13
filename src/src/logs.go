package main

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

var (
	Info     *log.Logger
	Warn     *log.Logger
	Err      *log.Logger
	fileName string = "machine.log.txt"
)

// MatchRes is the structure containing all pertinent information found
// while searching for a pattern in a log file.
// From MP0 best solution.
type MatchRes struct {
	MemberID       uint8
	LineNumber     int
	MatchedContent string
}

func InitLog() {
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(file)

	Info = log.New(
		file,
		"[INFO] ",
		log.Ldate|log.Ltime,
	)

	Warn = log.New(
		file,
		"[WARN] ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)

	Err = log.New(
		file,
		"[ERROR] ",
		log.Ldate|log.Ltime|log.Lshortfile,
	)
}

func printLogs(num_lines int) {
	file, err := os.Open(fileName)

	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	var txtlines []string

	for scanner.Scan() {
		txtlines = append(txtlines, scanner.Text())
	}

	// check if "get logs" with no -n was passed in,
	//    or num_lines is higher than the number of lines in the file
	if num_lines <= 0 || num_lines > len(txtlines) {
		num_lines = len(txtlines)
	}

	for i := len(txtlines) - num_lines; i < len(txtlines); i++ {
		fmt.Println(txtlines[i])
	}

	file.Close()
}

// Grep all files of every machine in mem's membership list
func (mem *Member) Grep(query string, local bool) {
	res := grepLocal(query, mem.memberID)
	for _, curr := range res {
		printGrep(curr)
	}

	if !local {
		for id, entry := range mem.membershipList {
			if id == mem.memberID {
				continue
			}

			mem.SendGrepRequest(entry.IPaddr, query)
		}
	}
}

func (mem *Member) SendGrepRequest(ip net.IP, query string) {
	// Encode query to send it
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(query)
	if err != nil {
		panic(err)
	}

	Send(ip.String()+":"+fmt.Sprint(Configuration.Service.port), GrepReq, b.Bytes())
}

func (mem *Member) HandleGrepResponse(queryBytes []byte) {
	// decode query
	b := bytes.NewBuffer(queryBytes)
	d := gob.NewDecoder(b)
	var resp MatchRes

	err := d.Decode(&resp)
	if err != nil {
		panic(err)
	}

	printGrep(resp)
}

// called when another process is requesting grep results
func (mem *Member) HandleGrepRequest(ip string, queryBytes []byte) {
	// decode desired query
	b := bytes.NewBuffer(queryBytes)
	d := gob.NewDecoder(b)
	var query string

	err := d.Decode(&query)
	if err != nil {
		panic(err)
	}

	// grep using the query on your local file
	res := grepLocal(query, mem.memberID)

	for _, curr := range res {
		// Encode result to send it back to the sender
		var b2 bytes.Buffer
		e := gob.NewEncoder(&b2)
		err2 := e.Encode(MatchRes{curr.MemberID, curr.LineNumber, curr.MatchedContent})
		if err2 != nil {
			panic(err2)
		}

		Send(ip+":"+fmt.Sprint(Configuration.Service.port), GrepResp, b2.Bytes())
	}
}

// call finder on the local file
func grepLocal(query string, memberID uint8) []MatchRes {
	//Find local log file
	matches := Finder(query, memberID)

	return matches
}

// Finder is the function that searches file at fileName, for pattern, using Go's
// Regex engine.
// From MP0 best solution.
func Finder(pattern string, memberID uint8) []MatchRes {
	retArr := make([]MatchRes, 0)
	//Create Regex from pattern
	regex, err := regexp.Compile(pattern)
	if err != nil {
		//Invalid Regex Pattern
		return make([]MatchRes, 0)
	}

	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		//Could not open file at fileName
		return make([]MatchRes, 0)
	}

	fileString := string(file)

	// Go through and find all lines that match pattern
	for lineIndex, line := range strings.Split(fileString, "\n") {
		if regex.MatchString(line) {
			newMatch := MatchRes{memberID, lineIndex, line}
			retArr = append(retArr, newMatch)
		}
	}
	return retArr
}

func printGrep(match MatchRes) {
	fmt.Printf("ID: %v | LineNo: %v | Text: %v", match.MemberID, match.LineNumber, match.MatchedContent)
	fmt.Println()
}
