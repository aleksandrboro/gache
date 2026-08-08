package main

import "fmt"

func main() {
	array := [3]int{1, 2, 3}
	slice := []int{1, 2, 3}

	ChangeArray(array)
	ChangeSlice(&slice)

	fmt.Println(array)
	fmt.Println(slice)
}

func ChangeSlice(slice *[]int) {
	*slice = append(*slice, 4)
}

func ChangeArray(slice [3]int) {
	for i := range slice {
		slice[i] *= 2
	}
}
