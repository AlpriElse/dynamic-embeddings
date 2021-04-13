package main

import (
	"strings"
)

func (m *Mapler) Maple(input string) error {

	// input is a line of text
	words := strings.Fields(input)
	for _, w := range words {
		m.Emit(w, "1")
	}
	return nil
}
