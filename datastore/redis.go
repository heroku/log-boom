package datastore

import (
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

// Datastore is the main interface into the db package.
type Datastore interface {
	Inserter
	HealthChecker
}

// Inserter is the interface for inserting records into the Datastore.
type Inserter interface {
	Insert(token string, lines []string) (int, error)
}

// HealthChecker is the interface for performming healthchecks against the Datastore.
type HealthChecker interface {
	Healthcheck() (bool, error)
}

// RedisDB is the redis implementation of the Datastore interface.
type RedisDB struct {
	p    *pool.Pool
	keep int
}

// NewRedisBackend creates an instance of RedisDB.
func NewRedisBackend(u *url.URL, keep, size int) (*RedisDB, error) {

	client, err := pool.NewCustom("tcp", u.Host, size, dialer(u.User))
	if err != nil {
		return nil, err
	}

	db := &RedisDB{
		p:    client,
		keep: keep,
	}

	return db, nil
}

// Insert inserts a batch into redis.
func (db *RedisDB) Insert(token string, lines []string) (int, error) {
	conn, err := db.p.Get()
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "Insert",
			"err": err,
		}).Error()
		return 0, err
	}
	defer db.p.Put(conn)

	if err := conn.Cmd("LPUSH", token, lines).Err; err != nil {
		log.WithFields(log.Fields{
			"at":  "Insert",
			"err": err,
		}).Error()
		return 0, err
	}

	if err := conn.Cmd("LTRIM", token, 0, db.keep).Err; err != nil {
		log.WithFields(log.Fields{
			"at":  "Insert",
			"err": err,
		}).Error()
		return len(lines), err
	}

	log.WithFields(log.Fields{
		"at":    "Insert",
		"lines": len(lines),
	}).Info("successfully stored logs in redis")
	return len(lines), nil
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
