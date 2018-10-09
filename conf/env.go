package conf

import (
	"fmt"
	"strings"
	"os"
	"github.com/jmoiron/sqlx"
	_"github.com/go-sql-driver/mysql"
	log4 "github.com/jeanphorn/log4go"
)

var MYSQL_MASTER_HOST string
var MYSQL_SLAVE_HOST  string
var MYSQL_PORT 	      string
var MYSQL_USERNAME    string
var MYSQL_PASSWORD	  string
var MYSQL_DB		  string
var LOG_FOR_ERROR	  string		// 放置非业务代码的错误的log
var LOG_FOR_LOGIC	  string		// 放置代码逻辑错误、运行参数、结果等的log
var ErrorLogger		  log4.Logger
var LogicLogger		  log4.Logger

func init() {
	LOG_FOR_ERROR = "/home/wwwlogs/php/ginder/php.log"
	LOG_FOR_LOGIC = "/home/wwwlogs/ginder/php.log"
	ErrorLogger = log4.NewDefaultLogger(log4.FINE)
	LogicLogger = log4.NewDefaultLogger(log4.FINE)
	ErrorLogger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(LOG_FOR_ERROR, true, true))
	LogicLogger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(LOG_FOR_LOGIC, true, true))
}

func DevStart() {
	MYSQL_MASTER_HOST = ""
	MYSQL_SLAVE_HOST = ""
	MYSQL_PORT = ""
	MYSQL_USERNAME = ""
	MYSQL_PASSWORD = ""
	MYSQL_DB = "passport"
}

func ProStart() {
	MYSQL_MASTER_HOST = strings.TrimSpace(os.Getenv("MYSQL_ETC1_MASTER_HOST"))
	MYSQL_SLAVE_HOST = strings.TrimSpace(os.Getenv("MYSQL_ETC1_SLAVE_HOST"))
	MYSQL_PORT = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PORT"))
	MYSQL_USERNAME = strings.TrimSpace(os.Getenv("MYSQL_ETC1_USERNAME"))
	MYSQL_PASSWORD = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PASSWORD"))
	MYSQL_DB = "passport"
}

func SqlMasterDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", MYSQL_USERNAME, MYSQL_PASSWORD,
		MYSQL_MASTER_HOST, MYSQL_PORT, MYSQL_DB)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		ErrorLogger.Error("mysql connect error: %s", err.Error())
	}

	return db
}

func SqlSlaveDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", MYSQL_USERNAME, MYSQL_PASSWORD,
		MYSQL_SLAVE_HOST, MYSQL_PORT, MYSQL_DB)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		ErrorLogger.Error("mysql connect error: %s", err.Error())
	}

	return db
}
