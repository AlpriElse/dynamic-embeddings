package main

import (
	"strconv"
)

func (j *Juice) Juice(key string, values []string) {
	count := 0
	for _, v := range values {
		if s, err := strconv.Atoi(v); err == nil {
			count += s
		}
	}
	j.Emit(key, strconv.Itoa(count))
}
