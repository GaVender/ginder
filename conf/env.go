package conf

import (
	"strings"
	"os"
)

var MYSQL_ETC1_MASTER_HOST string
var MYSQL_ETC1_SLAVE_HOST  string
var MYSQL_ETC1_PORT 	   string
var MYSQL_ETC1_USERNAME    string
var MYSQL_ETC1_PASSWORD	   string

func DevStart() {
	MYSQL_ETC1_MASTER_HOST = "127.0.0.1"
	MYSQL_ETC1_SLAVE_HOST = "127.0.0.1"
	MYSQL_ETC1_PORT = "3306"
	MYSQL_ETC1_USERNAME = "dad"
	MYSQL_ETC1_PASSWORD = "mom"
}

func ProStart() {
	MYSQL_ETC1_MASTER_HOST = strings.TrimSpace(os.Getenv("MYSQL_ETC1_MASTER_HOST"))
	MYSQL_ETC1_SLAVE_HOST = strings.TrimSpace(os.Getenv("MYSQL_ETC1_SLAVE_HOST"))
	MYSQL_ETC1_PORT = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PORT"))
	MYSQL_ETC1_USERNAME = strings.TrimSpace(os.Getenv("MYSQL_ETC1_USERNAME"))
	MYSQL_ETC1_PASSWORD = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PASSWORD"))
}
