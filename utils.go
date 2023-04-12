package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
)

const sampleConfig = "config.sample.json"

type clickhouseConfig struct {
	Servers        []string `json:"servers"`
	tlsServerName  string   `json:"tls_server_name"`
	tlsSkipVerify  bool     `json:"insecure_tls_skip_verify"`
	DownTimeout    int      `json:"down_timeout"`
	ConnectTimeout int      `json:"connect_timeout"`
}

// Config stores config data
type Config struct {
	Listen            string           `json:"listen"`
	Clickhouse        clickhouseConfig `json:"clickhouse"`
	FlushCount        int              `json:"flush_count"`
	FlushInterval     int              `json:"flush_interval"`
	CleanInterval     int              `json:"clean_interval"`
	RemoveQueryID     bool             `json:"remove_query_id"`
	DumpCheckInterval int              `json:"dump_check_interval"`
	DumpDir           string           `json:"dump_dir"`
	Debug             bool             `json:"debug"`
}

// ReadJSON - read json file to struct
func ReadJSON(fn string, v interface{}) error {
	file, err := os.Open(fn)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	return decoder.Decode(v)
}

// HasPrefix tests case insensitive whether the string s begins with prefix.
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && strings.ToLower(s[0:len(prefix)]) == strings.ToLower(prefix)
}

func readEnvInt(name string, value *int) {
	s := os.Getenv(name)
	if s != "" {
		v, err := strconv.Atoi(s)
		if err != nil {
			log.Printf("ERROR: Wrong %+v env: %+v\n", name, err)
		}
		*value = v
	}
}

func readEnvBool(name string, value *bool) {
	s := os.Getenv(name)
	if s != "" {
		v, err := strconv.ParseBool(s)
		if err != nil {
			log.Printf("ERROR: Wrong %+v env: %+v\n", name, err)
		}
		*value = v
	}
}

// ReadConfig init config data
func ReadConfig(configFile string) (Config, error) {
	cnf := Config{}
	err := ReadJSON(configFile, &cnf)
	if err != nil {
		log.Printf("INFO: Config file %+v not found. Used%+v\n", configFile, sampleConfig)
		err = ReadJSON(sampleConfig, &cnf)
		if err != nil {
			log.Printf("ERROR: read %+v failed\n", sampleConfig)
		}
	}

	readEnvBool("CLICKHOUSE_BULK_DEBUG", &cnf.Debug)
	readEnvInt("CLICKHOUSE_FLUSH_COUNT", &cnf.FlushCount)
	readEnvInt("CLICKHOUSE_FLUSH_INTERVAL", &cnf.FlushInterval)
	readEnvInt("CLICKHOUSE_CLEAN_INTERVAL", &cnf.CleanInterval)
	readEnvBool("CLICKHOUSE_REMOVE_QUERY_ID", &cnf.RemoveQueryID)
	readEnvInt("DUMP_CHECK_INTERVAL", &cnf.DumpCheckInterval)
	readEnvInt("CLICKHOUSE_DOWN_TIMEOUT", &cnf.Clickhouse.DownTimeout)
	readEnvInt("CLICKHOUSE_CONNECT_TIMEOUT", &cnf.Clickhouse.ConnectTimeout)
	readEnvBool("CLICKHOUSE_INSECURE_TLS_SKIP_VERIFY", &cnf.Clickhouse.tlsSkipVerify)

	serversList := os.Getenv("CLICKHOUSE_SERVERS")
	if serversList != "" {
		cnf.Clickhouse.Servers = strings.Split(serversList, ",")
	}
	log.Printf("use servers: %+v\n", strings.Join(cnf.Clickhouse.Servers, ", "))

	tlsServerName := os.Getenv("CLICKHOUSE_TLS_SERVER_NAME")
	if tlsServerName != "" {
		cnf.Clickhouse.tlsServerName = tlsServerName
	}

	return cnf, err
}

// Count not empty lines in txt file
func ListLen(list string) (int, error) {
	file, err := os.Open(list)
	if err != nil {
		log.Printf("INFO: problems with reading black-list [%+v]", err)
		return 0, err
	}
	defer file.Close()
	fileScanner := bufio.NewScanner(file)

	blacklist_len := 0
	for fileScanner.Scan() {
		if len(fileScanner.Text()) == 0 {
			continue
		}
		blacklist_len++
	}
	return blacklist_len, err
}

// Read txt file with and return list
func ReadList(blacklist string) ([]string, error) {
	bl_len, err := ListLen(blacklist)
	if err != nil {
		if err != nil {
			log.Printf("INFO: problems with reading black-list [%+v]", err)
			return nil, err
		}
	}

	file, err := os.Open(blacklist)
	if err != nil {
		log.Printf("INFO: problems with reading black-list [%+v]", err)
		return nil, err
	}

	defer file.Close()
	bl := make([]string, bl_len, 65535)

	i := 0
	fileScanner := bufio.NewScanner(file)
	for fileScanner.Scan() {
		user_name := fileScanner.Text()
		if len(user_name) == 0 {
			continue
		}
		bl[i] = strings.TrimSpace(user_name)
		i++
	}

	if err := fileScanner.Err(); err != nil {
		log.Printf("INFO: problems with reading black-list [%+v]", err)
	}
	return bl, nil
}
