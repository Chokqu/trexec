package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	seconds := 60
	if len(os.Args) > 1 {
		if s, err := strconv.Atoi(os.Args[1]); err == nil && s > 0 {
			seconds = s
		}
	}
	fmt.Printf("PID=%d sleeping for %ds\n", os.Getpid(), seconds)
	time.Sleep(time.Duration(seconds) * time.Second)
	fmt.Println("done")
}
