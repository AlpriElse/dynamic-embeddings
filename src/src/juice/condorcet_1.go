package main

func (j *Juice) Juice(key string, values []string) {
	count0 := 0
	count1 := 0

	for _, v := range values {
		if v == "0" {
			count0++
		} else {
			count1++
		}
	}

	if count1 >= count0 {
		j.Emit("", key)
	} else {
		j.Emit("", string(key[0])+string(key[3])+string(key[2])+string(key[1])+string(key[4]))
	}
}
