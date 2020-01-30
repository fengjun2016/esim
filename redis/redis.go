package redis

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/jukylin/esim/config"
	elog "github.com/jukylin/esim/log"
	"github.com/jukylin/esim/proxy"
)

var poolRedisOnce sync.Once
var onceRedisClient *redisClient

type redisClient struct {
	client *redis.Pool

	proxyConn []func() interface{}

	conf config.Config

	log elog.Logger

	proxyNum int

	proxyInses []interface{}
}


type Option func(c *redisClient)

type RedisClientOptions struct{}

func NewRedisClient(options ...Option) *redisClient {
	return newPoolRedis(options...)
}

func newPoolRedis(options ...Option) *redisClient {
	poolRedisOnce.Do(func() {

		onceRedisClient = &redisClient{
			proxyConn: make([]func () interface{}, 0),
		}

		for _, option := range options {
			option(onceRedisClient)
		}

		if onceRedisClient.conf == nil {
			onceRedisClient.conf = config.NewNullConfig()
		}

		if onceRedisClient.log == nil {
			onceRedisClient.log = elog.NewLogger()
		}

		onceRedisClient.proxyNum = len(onceRedisClient.proxyConn)
		if onceRedisClient.proxyNum > 0 {
			onceRedisClient.proxyInses = proxy.NewProxyFactory().
				GetInstances("redis", onceRedisClient.proxyConn...)
		}

		redisMaxActive := onceRedisClient.conf.GetInt("redis_max_active")
		if redisMaxActive == 0 {
			redisMaxActive = 500
		}

		redisMaxIdle := onceRedisClient.conf.GetInt("redis_max_idle")
		if redisMaxActive == 0 {
			redisMaxIdle = 100
		}

		redisIdleTimeout := onceRedisClient.conf.GetInt("redis_idle_time_out")
		if redisIdleTimeout == 0 {
			redisIdleTimeout = 600
		}

		redis_etc1_host := onceRedisClient.conf.GetString("redis_etc1_host")
		if redis_etc1_host == "" {
			redis_etc1_host = "0.0.0.0"
		}
		redis_etc1_port := onceRedisClient.conf.GetString("redis_etc1_post")
		if redis_etc1_port == "" {
			redis_etc1_port = "6379"
		}

		redis_etc1_password := onceRedisClient.conf.GetString("redis_etc1_password")

		redis_read_time_out := onceRedisClient.conf.GetInt64("redis_read_time_out")
		if redis_read_time_out == 0 {
			redis_read_time_out = 300
		}

		redis_write_time_out := onceRedisClient.conf.GetInt64("redis_write_time_out")
		if redis_write_time_out == 0 {
			redis_write_time_out = 300
		}

		redis_conn_time_out := onceRedisClient.conf.GetInt64("redis_conn_time_out")
		if redis_conn_time_out == 0 {
			redis_conn_time_out = 300
		}

		onceRedisClient.client = &redis.Pool{
			MaxIdle:     redisMaxIdle,
			MaxActive:   redisMaxActive,
			IdleTimeout: time.Duration(redisIdleTimeout) * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", redis_etc1_host+":"+redis_etc1_port,
					redis.DialReadTimeout(time.Duration(redis_read_time_out)*time.Millisecond),
					redis.DialWriteTimeout(time.Duration(redis_write_time_out)*time.Millisecond),
					redis.DialConnectTimeout(time.Duration(redis_conn_time_out)*time.Millisecond))

				if err != nil {
					onceRedisClient.log.Panicf("redis.Dial err: %s", err.Error())
					return nil, err
				}
				if redis_etc1_password != "" {
					if _, err := c.Do("AUTH", redis_etc1_password); err != nil {
						c.Close()
						onceRedisClient.log.Panicf("redis.AUTH err: %s", err.Error())
						return nil, err
					}
				}
				// 选择db
				c.Do("SELECT", 0)

				if onceRedisClient.conf.GetBool("debug") == true {
					c = redis.NewLoggingConn(
						c, log.New(os.Stdout, "",
							log.Ldate|log.Ltime|log.Lshortfile), "")
				}
				return c, nil
			},
		}

		if onceRedisClient.conf.GetString("runmode") == "pro" {
			//conn success ？
			rc := onceRedisClient.client.Get()
			rc.Close()
		}
		onceRedisClient.log.Infof("[redis] init success %s : %s", redis_etc1_host, redis_etc1_port)
	})

	return onceRedisClient
}

func (RedisClientOptions) WithConf(conf config.Config) Option {
	return func(r *redisClient) {
		r.conf = conf
	}
}

func (RedisClientOptions) WithLogger(log elog.Logger) Option {
	return func(r *redisClient) {
		r.log = log
	}
}

func (RedisClientOptions) WithProxy(proxyConn ...func() interface{}) Option {
	return func(r *redisClient) {
		r.proxyConn = append(r.proxyConn, proxyConn...)
	}
}

//使用原生redisgo
func (r *redisClient) GetRedisConn() redis.Conn {

	rc := r.client.Get()

	return rc
}

func (r *redisClient) GetCtxRedisConn() ContextConn {

	rc := r.client.Get()
	mediatorProxy := NewMediatorProxy()
	mediatorProxy.NextProxy(rc)

	var firstProxy ContextConn
	if r.proxyNum > 0 {
		firstProxy = r.proxyInses[len(r.proxyInses) - 1].(ContextConn)
		firstProxy.(RedisProxy).NextProxy(mediatorProxy)
	}else{
		firstProxy = mediatorProxy
	}

	return firstProxy
}
