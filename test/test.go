package main

import (
	"binary"
	"fmt"
	"log"
	"math"
	"os/exec"
	"runtime"
)

func Float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

// | grep \"8123\" | wc -l
func main() {
	if runtime.GOOS == "windows" {
		fmt.Println("Can't Execute this on a windows machine")
	} else {
		out1, err := exec.Command("bash", "-c", "ss -p | grep \"8123\" | wc -l").Output()

		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(out1))
	}
}
