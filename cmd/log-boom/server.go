package main

import (
	"net/http"
	"net/url"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	ds "github.com/voidlock/log-boom/datastore"
)

// DefaultRedisPoolSize is the default pool size (defaults to 4).
const DefaultRedisPoolSize = 4

type env struct {
	db ds.Datastore
}

func (e *env) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := http.StatusNoContent

	ok, err := e.db.Healthcheck()
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "healthcheck",
			"err": err,
		}).Error("unable to healthcheck datastore")
		status = http.StatusServiceUnavailable
	}
	if !ok {
		log.WithFields(log.Fields{
			"at": "healthcheck",
		}).Error("healthcheck failed")
		status = http.StatusServiceUnavailable
	}

	log.WithFields(log.Fields{
		"at": "healthcheck",
		"ok": ok,
	}).Debug("health checked")
	w.WriteHeader(status)
}

func (e *env) logsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	listen := os.Getenv("LISTEN")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	url, err := url.Parse(os.Getenv("REDIS_URL"))
	if err != nil || url.Scheme != "redis" {
		log.Fatal("$REDIS_URL must be set and valid")
	}
	size, err := strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	if err != nil {
		size = DefaultRedisPoolSize
	}
	db, err := ds.NewRedisBackend(url, size)
	if err != nil {
		log.Fatal(err)
	}

	e := &env{db}

	http.HandleFunc("/healthcheck", e.healthHandler)
	http.HandleFunc("/logs", e.logsHandler)

	if err := http.ListenAndServe(listen+":"+port, nil); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("unable to start server")
	}
}
