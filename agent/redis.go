package agent

import (
	"fmt"
	"io"

	"github.com/garyburd/redigo/redis"
	"github.com/gogo/protobuf/proto"
	"github.com/vektra/cypress"
)

type RedisOutput struct {
	conn     redis.Conn
	host     string
	listName string
	feeder   chan []byte
}

func (r *RedisOutput) Start(host, list string) error {
	c, err := redis.Dial("tcp", host)

	if err != nil {
		return err
	}

	r.host = host
	r.conn = c
	r.listName = list

	r.feeder = make(chan []byte)

	go r.process()

	return nil
}

func (r *RedisOutput) process() {
	for {
		data := <-r.feeder

	retry:
		_, err := r.conn.Do("lpush", r.listName, data)

		if err != nil {
			if err == io.EOF {
				c, err := redis.Dial("tcp", r.host)

				if err != nil {
					fmt.Printf("Error reconnecting to redis: %s\n", err)
					return
				}

				r.conn = c
				goto retry
			} else {
				fmt.Printf("Error writing to redis: %s\n", err)
			}
		}
	}
}

func (r *RedisOutput) Read(m *cypress.Message) error {
	data, err := proto.Marshal(m)

	if err != nil {
		return err
	}

	r.feeder <- data

	return nil
}

type RedisInput struct {
	conn     redis.Conn
	host     string
	listName string
	recv     cypress.Receiver
	active   bool
}

func (r *RedisInput) Init(host, list string, rc cypress.Receiver) error {
	c, err := redis.Dial("tcp", host)

	if err != nil {
		return err
	}

	r.host = host
	r.conn = c
	r.recv = rc
	r.listName = list
	r.active = true

	return nil
}

func (r *RedisInput) Close() {
	r.active = false
	r.conn.Close()
}

func (r *RedisInput) Start() error {
	for {
		vals, err := redis.Values(r.conn.Do("brpop", r.listName, 0))

		if err != nil {
			if err == io.EOF {
				if !r.active {
					return nil
				}

				c, err := redis.Dial("tcp", r.host)

				if err != nil {
					return err
				}

				r.conn = c
				continue
			}

			return err
		}

		data, err := redis.Bytes(vals[1], err)

		m := &cypress.Message{}

		err = proto.Unmarshal(data, m)

		if err != nil {
			fmt.Printf("Error reading message: %s\n", err)
			continue
		}

		r.recv.Read(m)
	}

	return nil
}
