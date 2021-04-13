package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

type IMaple interface {
	// processes value (map needs no key) and calls Emit
	Maple(input string) error
}

// Mapler is a struct that allows implementing Maple method by user with receiver of Mapler type
type Mapler struct {
}

// Emit writes key,value pair to stdout
func (m *Mapler) Emit(key string, value string) {
	fmt.Println(key + "," + value)
}

func processInputFile(inputFilePath string) error {
	file, err := os.Open(inputFilePath)
	if err != nil {
		log.Fatal(err)
		log.Println("Error opening file")
	}
	defer file.Close()

	// create Mapler object
	var mapler IMaple
	maplerObj := Mapler{}
	mapler = &maplerObj

	scanner := bufio.NewScanner(file)

	// send lines of text to Maple
	// todo: modify this to process 10-20 lines at once as given in spec
	for scanner.Scan() {
		mapler.Maple(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		log.Println("Error scanning tokens")
	}

	return err
}

func main() {
	if len(os.Args) < 2 {
		log.Println("Insufficient arguments to maple")
		return
	}
	processInputFile(os.Args[1])
}
