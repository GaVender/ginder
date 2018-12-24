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
	"gopkg.in/mgo.v2"
	"time"
)

var mysql_master_host 		string
var mysql_slave_host  		string
var mysql_port 	      		string
var mysql_username    		string
var mysql_password	  		string
var mysql_db		  		string
var redis_master_host 		string
var redis_slave_host  		string
var redis_port		  		string
var redis_password	  		string
var redis_db		  		int
var redis_pool_size   		int
var mongo_host 				[]string
var mongo_user				string
var mongo_password			string
var mongo_timeout			time.Duration
var mongo_pool_limit		int
var mongo_session    		*mgo.Session
var log_for_error	  		string		// 放置非业务代码的错误的log路径
var log_for_logic	  		string		// 放置代码逻辑错误、运行参数、结果等的log路径
var error_logger	  		log4.Logger
var logic_logger	  		log4.Logger


func Start(_log_for_error string, _log_for_logic string) {
	if _log_for_error == "" {
		panic("请设置好异常错误日志路径")
	}

	if _log_for_logic == "" {
		panic("请设置好逻辑错误日志路径")
	}

	log_for_error 		= strings.TrimSpace(_log_for_error)
	log_for_logic 		= strings.TrimSpace(_log_for_logic)
	mysql_master_host 	= strings.TrimSpace(os.Getenv("MYSQL_MASTER_HOST"))
	mysql_slave_host 	= strings.TrimSpace(os.Getenv("MYSQL_SLAVE_HOST"))
	mysql_port 			= strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	mysql_username 		= strings.TrimSpace(os.Getenv("MYSQL_USERNAME"))
	mysql_password 		= strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	mysql_db 			= strings.TrimSpace(os.Getenv("MYSQL_DB"))
	redis_master_host 	= strings.TrimSpace(os.Getenv("REDIS_MASTER_HOST"))
	redis_slave_host 	= strings.TrimSpace(os.Getenv("REDIS_SLAVE_HOST"))
	redis_port 			= strings.TrimSpace(os.Getenv("REDIS_PORT"))
	redis_password 		= strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redis_db, _ 		= strconv.Atoi(os.Getenv("REDIS_DB"))
	redis_pool_size, _ 	= strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	mongo_host 			= []string{strings.TrimSpace(os.Getenv("MONGO_HOST"))}
	mongo_user 			= strings.TrimSpace(os.Getenv("MONGO_USER"))
	mongo_password 		= strings.TrimSpace(os.Getenv("MONGO_PASSWORD"))
	mongo_timeout 		= time.Second * 2
	mongo_pool_limit, _ = strconv.Atoi(os.Getenv("MONGO_POOL_LIMIT"))

	/*
		设置好日志配置，一个是系统级别错误，一个是业务逻辑错误
	*/
	error_logger = log4.NewDefaultLogger(log4.FINE)
	logic_logger = log4.NewDefaultLogger(log4.FINE)
	error_logger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(log_for_error, true, true))
	logic_logger.AddFilter("file", log4.FINE, log4.NewFileLogWriter(log_for_logic, true, true))

	/*
		启动mysql、redis、mongo配置
	*/
	fmt.Println("mysql 主库启动......")
	SqlMasterDb()
	fmt.Println("mysql 从库启动......")
	SqlSlaveDb()
	fmt.Println("redis 主库启动......")
	RedisMaster()
	fmt.Println("redis 从库启动......")
	RedisSlave()
	fmt.Println("mongo 启动......")
	MongoSession()
}

func SqlMasterDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysql_username, mysql_password,
		mysql_master_host, mysql_port, mysql_db)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		LoggerError().Error("mysql connect error: %s", err.Error())
		panic("mysql connect error")
	}

	return db
}

func SqlSlaveDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysql_username, mysql_password,
		mysql_slave_host, mysql_port, mysql_db)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		LoggerError().Error("mysql connect error: %s", err.Error())
		panic("mysql connect error")
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

func mongoSessionFactory() *mgo.Session {
	var err error

	dialInfo := &mgo.DialInfo{
		Addrs:     mongo_host,
		Username:  mongo_user,
		Password:  mongo_password,
		Timeout:   mongo_timeout,
		Direct:    false,
		PoolLimit: mongo_pool_limit,
	}

	mongo_session, err = mgo.DialWithInfo(dialInfo)

	if err != nil {
		LoggerError().Error("mongo connect error : %s", err.Error())
		panic("mongo connect error")
	} else {
		mongo_session.SetMode(mgo.Eventual, true)
		return mongo_session
	}
}

func MongoSession() *mgo.Session {
	if mongo_session == nil {
		mongo_session = mongoSessionFactory()
	}

	return mongo_session.Copy()
}