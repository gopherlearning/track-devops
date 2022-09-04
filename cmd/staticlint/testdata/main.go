package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("123")
	os.Exit(0) // assert "using os Exit!"
}
