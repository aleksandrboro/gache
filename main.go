package main

import "fmt"

func main() {
	m := make(map[string][]byte)
	v := m["a"]
	if v == nil {
		v = []byte("asd")
	}

	fmt.Println(m)
}
