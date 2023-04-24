package main

import (
	"flag"
	"log"
	"os"
)

var version = "unknown"
var date = "unknown"
var admins []string
var dmicp_password string

func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	configFile := flag.String("config", "config.json", "config file (json)")

	flag.Parse()

	if flag.Arg(0) == "version" {
		log.Printf("clickhouse-bulk ver. %+v (%+v)\n", version, date)
		return
	}

	cnf, err := ReadConfig(*configFile)
	if err != nil {
		log.Fatalf("ERROR: %+v\n", err)
	}
	admins = cnf.Admins
	dmicp_password = cnf.DmicpPass
	RunServer(cnf)
}
