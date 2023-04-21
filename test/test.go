package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
)

// | grep \"8123\" | wc -l
func main() {
	if runtime.GOOS == "windows" {
		fmt.Println("Can't Execute this on a windows machine")
	} else {
		out1, err := exec.Command("bash", "-c", "ss -p | grep \"8123\" | wc -l").Output()

		if err != nil {
			log.Fatal(err)
		}
		str := regexp.MustCompile(`[^a-zA-Z0-9 ]+`).ReplaceAllString(string(out1), "")
		fmt.Println(strconv.ParseFloat(string(str), 64))
	}
}
