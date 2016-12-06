package datastore

import (
	"net/url"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

// Datastore is the main interface into the db package.
type Datastore interface {
	Inserter
	HealthChecker
}

// Batch is a batch of logs to insert.
type Batch struct {
}

// Inserter is the interface for inserting records into the Datastore.
type Inserter interface {
	Insert(batch Batch)
}

// HealthChecker is the interface for performming healthchecks against the Datastore.
type HealthChecker interface {
	Healthcheck() (bool, error)
}

// RedisDB is the redis implementation of the Datastore interface.
type RedisDB struct {
	p *pool.Pool
}

// NewRedisBackend creates an instance of RedisDB.
func NewRedisBackend(u *url.URL, size int) (*RedisDB, error) {

	client, err := pool.NewCustom("tcp", u.Host, size, dialer(u.User))
	if err != nil {
		return nil, err
	}

	return &RedisDB{client}, nil
}

// Insert inserts a batch into redis.
func (db *RedisDB) Insert(batch Batch) {

}

// Healthcheck performs a PING against redis.
func (db *RedisDB) Healthcheck() (bool, error) {
	pong, err := db.p.Cmd("PING").Str()
	if pong == "PONG" {
		return true, nil
	}
	return false, err
}

func dialer(user *url.Userinfo) pool.DialFunc {
	if user == nil {
		return redis.Dial
	}

	secret, ok := user.Password()
	if !ok {
		return redis.Dial
	}

	return func(network, address string) (*redis.Client, error) {
		client, err := redis.Dial(network, address)
		if err != nil {
			return nil, err
		}

		if err = client.Cmd("AUTH", secret).Err; err != nil {
			client.Close()
			return nil, err
		}
		return client, nil
	}
}
