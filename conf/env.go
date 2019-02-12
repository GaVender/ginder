package conf

import (
	"fmt"
	"strings"
	"os"
	"time"
	"strconv"

	log4 "ginder/log/log4go"

	"github.com/jmoiron/sqlx"
	_"github.com/go-sql-driver/mysql"
	"gopkg.in/redis.v5"
	"gopkg.in/mgo.v2"
)

type LogProvider interface {
	Close()
	LogInfo(head, content string)
	LogWarn(head, content string)
	LogError(head, content string)
}

var mysqlMasterHost 		string
var mysqlSlaveHost  		string
var mysqlPort 	      		string
var mysqlUsername    		string
var mysqlPwd	  			string
var mysqlDb		  			string
var redisMasterHost 		string
var redisSlaveHost  		string
var redisPort		  		string
var redisPwd	  			string
var redisDb		  			int
var redisPoolSize   		int
var mongoHost 				[]string
var mongoUser				string
var mongoPwd				string
var mongoTimeout			int
var mongoPoolLimit			int
var mongoSession    		*mgo.Session
var loggerForLogic 			LogProvider
var loggerForError 			LogProvider

var ErrorLogStart    = false
var LogicLogStart    = false
var MysqlMasterStart = false
var MysqlSlaveStart  = false
var RedisMasterStart = false
var RedisSlaveStart  = false
var MongoStart 		 = false


func init() {
	start()
}

func start() {
	/*
		参数初始化
	*/
	mysqlMasterHost	  	= strings.TrimSpace(os.Getenv("MYSQL_MASTER_HOST"))
	mysqlSlaveHost 	  	= strings.TrimSpace(os.Getenv("MYSQL_SLAVE_HOST"))
	mysqlPort 		  	= strings.TrimSpace(os.Getenv("MYSQL_PORT"))
	mysqlUsername 	  	= strings.TrimSpace(os.Getenv("MYSQL_USERNAME"))
	mysqlPwd 		  	= strings.TrimSpace(os.Getenv("MYSQL_PASSWORD"))
	mysqlDb 		  	= strings.TrimSpace(os.Getenv("MYSQL_DB"))
	redisMasterHost   	= strings.TrimSpace(os.Getenv("REDIS_MASTER_HOST"))
	redisSlaveHost 	  	= strings.TrimSpace(os.Getenv("REDIS_SLAVE_HOST"))
	redisPort 		  	= strings.TrimSpace(os.Getenv("REDIS_PORT"))
	redisPwd 		  	= strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	redisDb, _ 		  	= strconv.Atoi(os.Getenv("REDIS_DB"))
	redisPoolSize, _  	= strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	mongoHost 		  	= []string{strings.TrimSpace(os.Getenv("MONGO_HOST"))}
	mongoUser 		  	= strings.TrimSpace(os.Getenv("MONGO_USER"))
	mongoPwd 		  	= strings.TrimSpace(os.Getenv("MONGO_PASSWORD"))
	mongoTimeout, _   	= strconv.Atoi(os.Getenv("MONGO_TIMEOUT"))
	mongoPoolLimit, _	= strconv.Atoi(os.Getenv("MONGO_POOL_LIMIT"))

	/*
		日志配置，一个是系统级别错误，一个是业务逻辑错误
	*/
	loggerForError = log4.GetErrorLogger()
	ErrorLogStart  = true
	loggerForLogic = log4.GetLogicLogger()
	LogicLogStart  = true

	/*
		启动mysql、redis、mongo配置
	*/
	SqlMasterDb()
	SqlSlaveDb()
	RedisMaster()
	RedisMasterStart = true
	RedisSlave()
	RedisSlaveStart = true
	MongoSession()
}

func GetErrorLogger() LogProvider {
	return loggerForError
}

func GetLogicLogger() LogProvider {
	return loggerForLogic
}

func SqlMasterDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysqlUsername, mysqlPwd,
		mysqlMasterHost, mysqlPort, mysqlDb)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		loggerForError.LogError("master mysql", fmt.Sprintf("connect error: %s", err.Error()))
		panic("mysql connect error")
	} else {
		MysqlMasterStart = true
	}

	return db
}

func SqlSlaveDb() *sqlx.DB {
	dns := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", mysqlUsername, mysqlPwd,
		mysqlSlaveHost, mysqlPort, mysqlDb)
	db, err := sqlx.Open("mysql", dns)

	if err != nil {
		loggerForError.LogError("slave mysql", fmt.Sprintf("connect error: %s", err.Error()))
		panic("mysql connect error")
	} else {
		MysqlSlaveStart = true
	}

	return db
}

func redisFactory(name string) *redis.Client {
	host := ""

	if "master" == name {
		host = redisMasterHost
	} else {
		host = redisSlaveHost
	}

	return redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", host, redisPort),
		Password:    redisPwd,
		DB:          redisDb,
		PoolSize:    redisPoolSize,
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

func mongoSessionFactory() *mgo.Session {
	var err error

	dialInfo := &mgo.DialInfo{
		Addrs:     mongoHost,
		Username:  mongoUser,
		Password:  mongoPwd,
		Timeout:   time.Second * time.Duration(mongoTimeout),
		Direct:    false,
		PoolLimit: mongoPoolLimit,
	}

	mongoSession, err = mgo.DialWithInfo(dialInfo)

	if err != nil {
		loggerForError.LogError("mongo", fmt.Sprintf("connect error: %s", err.Error()))
		panic("mongo connect error")
	} else {
		MongoStart = true
		mongoSession.SetMode(mgo.Eventual, true)
		return mongoSession
	}
}

func MongoSession() *mgo.Session {
	if mongoSession == nil {
		mongoSession = mongoSessionFactory()
	}

	return mongoSession.Copy()
}