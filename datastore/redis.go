package datastore

import (
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
)

// RedisDB is the redis implementation of the Datastore interface.
type RedisDB struct {
	p    *pool.Pool
	keep int
}

// NewInRedis creates an instance of RedisDB.
func NewInRedis(u *url.URL, keep, size int) (*RedisDB, error) {

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

// List performs LRANGE agains redis.
func (db *RedisDB) List(token string) ([]string, error) {
	conn, err := db.p.Get()
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "List",
			"err": err,
		}).Error()
		return nil, err
	}
	defer db.p.Put(conn)

	exists, err := conn.Cmd("EXISTS", token).Int()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, ErrNoSuchToken
	}

	lines, err := conn.Cmd("LRANGE", token, 0, -1).List()
	if err != nil {
		return nil, err
	}

	return lines, nil
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
