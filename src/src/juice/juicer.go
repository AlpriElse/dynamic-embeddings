package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// Interace for Juice
type IJuice interface {
	Juice(key string, values []string)
}

// run partitioner on keys
// start maple tasks for each key
type Juice struct {
	keys   []string
	values []string
}

// Emit a key/value to juicer
func (j *Juice) Emit(key string, value string) {
	j.keys = append(j.keys, key)
	j.values = append(j.values, value)
}

// Generate a new file from keys and values emitted
func (j *Juice) GenerateFile() map[string]string {
	filesGen := make(map[string]string)
	// for every key in list of keys
	for i, key := range j.keys {
		// concatenate values[i] + "\n"
		currStr, ok := filesGen[key]
		if !ok {
			fmt.Println("No key found")
		}
		filesGen[key] = currStr + j.values[i] + "\n"
	}
	return filesGen
}

// Save to files based on key
func (j *Juice) Save(prefix string) {
	// Parse keys and values into a single string
	// Every key is mapped to a string with new value
	newData := j.GenerateFile()
	for k, v := range newData {
		SaveToFile(prefix+"_"+k, v)
	}
}

// Save all keys to one file
func (j *Juice) SaveAllOutput(outputFname string) {
	newData := j.GenerateFile()
	out := ""
	for k, v := range newData {
		out = out + k + "\t" + v
	}
	SaveToFile(outputFname, out)
}

// Save string to file
func SaveToFile(fname string, value string) {
	// truncate end of string
	/*
	   if len(value) > 2 {
	           value = value[0:len(value)-3]
	   }*/
	fileFlags := os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(fname, fileFlags, 0777)
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	if _, err := file.WriteString(value); err != nil {
		fmt.Println(err)
	}
}

// Read stdin for some fruits to juice
func ReadStdin() *bytes.Buffer {
	stdinReader := bufio.NewReader(os.Stdin)
	fruit := bytes.NewBuffer(make([]byte, 0))
	bytes, _ := ioutil.ReadAll(stdinReader)
	fruit.Write(bytes)
	return fruit
}

func main() {
	// Get filename
	dir := "."
	key := "no_key"
	pref := "no_prefix"
	if len(os.Args) > 2 {
		dir = os.Args[1]
		pref = os.Args[2]
		key = os.Args[3]
	}

	// Listen to IPC or stdin for juice data
	var juicer IJuice
	juicerObj := Juice{keys: make([]string, 0), values: make([]string, 0)}

	juicer = &juicerObj
	fruits := ReadStdin()
	// Split string by new line
	vals := strings.Split(fruits.String(), "\n")
	// Run Juice function
	juicer.Juice(key, vals)
	// Save juice result to file
	juicerObj.SaveAllOutput(dir + "/" + pref + "_" + key)
	return
}
