package redis

import (
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"github.com/henrylee2cn/faygo"
	"os"
	"time"
	"wallet/config"
)

type Redis struct {
	conn  redis.Conn
	queue chan string //需要存入的数据队列
}

var redisCient *redis.Pool

var host string

func init() {
	host = config.GetConfig("redis", "host").String()
	MaxIdle, err := config.GetConfig("redis", "MaxIdle").Int()
	if err != nil {
		faygo.Info("获取配置出错")
		os.Exit(2)
	}
	MaxActive, err := config.GetConfig("redis", "MaxActive").Int()
	if err != nil {
		faygo.Info("获取配置出错")
		os.Exit(2)
	}
	redisCient = &redis.Pool{
		MaxIdle:     MaxIdle,
		MaxActive:   MaxActive,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host)
			if err != nil {
				return nil, err
			}
			// 选择db
			c.Do("SELECT", "1")
			return c, nil
		},
	}
	redisCient.Get()
	redisCient.Dial()
	return
}

func New() (r *Redis) {
	r = new(Redis)
	r.conn = redisCient.Get()
	q := make(chan string, max_queue_len)
	r.queue = q
	return r
}

//将对象放回连接池子中
func (r *Redis) put() {
	r.conn.Close()
}

func (r *Redis) Get(key string) (reply string, err error) {
	reply, err = redis.String(r.conn.Do("GET", key))
	defer r.put()
	return
}

func (r *Redis) Set(key string, value interface{}, expire ...interface{}) (reply interface{}, err error) {
	if len(expire) == 1 {
		reply, err = r.conn.Do("SET", key, value, "EX", expire[0])
	} else {
		reply, err = r.conn.Do("SET", key, value)
	}
	defer r.put()
	return
}

func (r *Redis) Del(key string) (reply string, err error) {
	reply, err = redis.String(r.conn.Do("DEL", key))
	defer r.put()
	return
}

//格式化输出的结果
func Tofloat(reply interface{}, reply_err error) (f float64, err error) {
	f, err = redis.Float64(reply, reply_err)
	return
}

func newConsumerRedis() (r *Redis, err error) {
	conn, err := redis.Dial("tcp", host)
	if err != nil {
		return
	}
	conn.Do("SELECT", "1")
	r = new(Redis)
	r.conn = conn
	q := make(chan string, max_queue_len)
	r.queue = q
	return
}

const max_queue_len = 10 //默认channel的容量

//选择库
func (r *Redis) Select(dbname string) {

}

func (r *Redis) Lpush(key, data string) error {
	_, err := r.conn.Do("lpush", key, data)
	defer r.put()
	return err
}

func (r *Redis) Rpush(key, data string) error {
	_, err := r.conn.Do("rpush", key, data)
	defer r.put()
	return err
}

//生产者
func (r *Redis) Product(key string, data interface{}) error {
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	str := string(res)
	err = r.Lpush(key, str)
	if err != nil {
		return err
	}
	return nil
}

//重新放回队列的位置
func (r *Redis) Replay(key string, data interface{}) error {
	res, err := json.Marshal(data)
	if err != nil {
		return err
	}
	str := string(res)
	err = r.Rpush(key, str)
	if err != nil {
		return err
	}
	return nil
}

//消费者
func Consumer(key string, chan_len ...int) chan string {
	redisQueueConn, err := newConsumerRedis()
	if err != nil {
		return nil
	}
	c_len := max_queue_len
	if len(chan_len) > 0 {
		c_len = chan_len[0]
	}
	if redisQueueConn.queue == nil {
		redisQueueConn.queue = make(chan string, c_len)
	}
	//启动一个协程不断的去从队列中获取数据。并且将数据放到channel里面
	go func(c chan string, key string) {
		for {
			res, err := redis.StringMap(redisQueueConn.conn.Do("brpop", key, 0))
			if err == nil {
				c <- res[key]
			} else {
				faygo.Debug(56788)
			}
		}

	}(redisQueueConn.queue, key)
	return redisQueueConn.queue
}
