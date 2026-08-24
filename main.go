package main

import "fmt"

func main() {
	m := make(map[string][]byte)

	m[" "] = []byte("")

	fmt.Println(len(m))
}
