package conf

import (
	"fmt"
	"strings"
	"os"
	"github.com/jmoiron/sqlx"
	_"github.com/go-sql-driver/mysql"
	log4 "github.com/jeanphorn/log4go"
	"strconv"
	"gopkg.in/redis.v5"
)

var mysql_master_host string
var mysql_slave_host  string
var mysql_port 	      string
var mysql_username    string
var mysql_password	  string
var mysql_db		  string
var redis_master_host string
var redis_slave_host  string
var redis_port		  string
var redis_password	  string
var redis_db		  int
var redis_pool_size   int
var log_for_error	  string		// 放置非业务代码的错误的log
var log_for_logic	  string		// 放置代码逻辑错误、运行参数、结果等的log
var error_logger	  log4.Logger
var logic_logger	  log4.Logger

func init() {
	log_for_error = "/home/wwwlogs/php/ginder/php.log"
	log_for_logic = "/home/wwwlogs/ginder/php.log"
	error_logger = log4.NewDefaultLogger(log4.FINE)
	logic_logger = log4.NewDefaultLogger(log4.FINE)
	error_logger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(log_for_error, true, true))
	logic_logger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(log_for_logic, true, true))
}

func DevStart() {
	mysql_master_host = ""
	mysql_slave_host = ""
	mysql_port = "3306"
	mysql_username = ""
	mysql_password = ""
	mysql_db = "passport"
	redis_master_host = "127.0.0.1"
	redis_slave_host = "127.0.0.1"
	redis_port = "6379"
	redis_password = ""
	redis_db = 0
	redis_pool_size = 10
}

func ProStart() {
	mysql_master_host = strings.TrimSpace(os.Getenv("MYSQL_ETC1_MASTER_HOST"))
	mysql_slave_host = strings.TrimSpace(os.Getenv("MYSQL_ETC1_SLAVE_HOST"))
	mysql_port = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PORT"))
	mysql_username = strings.TrimSpace(os.Getenv("MYSQL_ETC1_USERNAME"))
	mysql_password = strings.TrimSpace(os.Getenv("MYSQL_ETC1_PASSWORD"))
	mysql_db = "passport"
	redis_master_host = strings.TrimSpace(os.Getenv("redis_master_host"))
	redis_slave_host = strings.TrimSpace(os.Getenv("redis_slave_host"))
	redis_port = strings.TrimSpace(os.Getenv("redis_port"))
	redis_password = strings.TrimSpace(os.Getenv("redis_password"))
	redis_db, _ = strconv.Atoi(os.Getenv("redis_db"))
	redis_pool_size, _ = strconv.Atoi(os.Getenv("redis_pool_size"))
}

func SqlMasterDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysql_username, mysql_password,
		mysql_master_host, mysql_port, mysql_db)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		LoggerError().Error("mysql connect error: %s", err.Error())
	}

	return db
}

func SqlSlaveDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysql_username, mysql_password,
		mysql_slave_host, mysql_port, mysql_db)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		LoggerError().Error("mysql connect error: %s", err.Error())
	}

	return db
}

func redisFactory(name string) *redis.Client {
	host := ""

	if "master" == name {
		host = redis_master_host
	} else {
		host = redis_slave_host
	}

	return redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", host, redis_port),
		Password:    redis_password,
		DB:          redis_db,
		PoolSize:    redis_pool_size,
	})
}

func getRedis(name string) *redis.Client {
	return redisFactory(name)
}

func RedisMaster() *redis.Client {
	return getRedis("master")
}

func RedisSlave() *redis.Client {
	return getRedis("slave")
}

func LoggerError() log4.Logger {
	return error_logger
}

func LoggerLogic() log4.Logger {
	return logic_logger
}