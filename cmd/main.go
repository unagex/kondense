package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	// get the containers running alongside the main app
	for {
		time.Sleep(time.Second)
		// env :=
		fmt.Println(os.Getenv("HOSTNAME"))
		fmt.Println("oula")
	}
}
