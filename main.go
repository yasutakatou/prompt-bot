package main

import (
	"fmt"
	"log"
	"os"
	"unicode/utf8"
)

func main() {
	indexWord := "文字列"
	//indexWord := "\u6587\u5b57\u5217"

	file, err := os.Create("test")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fmt.Println(indexWord)
	fmt.Println(utf8.ValidString(indexWord))
	fmt.Fprintln(file, indexWord+"\t"+"prompt"+"\t"+"result"+"\t"+"t")

	valid := "Hello, 世界"
	invalid := string([]byte{0xff, 0xfe, 0xfd})

	// fmt.Println(utf8.ValidString(valid))
	// fmt.Println(utf8.ValidString(invalid))
	// Output:
	// true
	// false
}
