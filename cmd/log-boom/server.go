package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/heroku/log-boom/auth"
	ds "github.com/heroku/log-boom/datastore"
	"github.com/heroku/log-boom/syslog"
	"goji.io"
	"goji.io/pat"
)

const (
	// DefaultRedisPoolSize is the default pool size (defaults to 4).
	DefaultRedisPoolSize = 4

	// DefaultBufferSize is the default size of the ring bugger in log lines.
	DefaultBufferSize = 1500
)

type env struct {
	db ds.Datastore
}

func (e *env) healthHandler(w http.ResponseWriter, r *http.Request) {
	ok, err := e.db.Healthcheck()
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "healthcheck",
			"err": err,
		}).Error("unable to healthcheck datastore")

		http.Error(w, http.StatusText(503), 503)
		return
	}

	if !ok {
		log.WithFields(log.Fields{
			"at": "healthcheck",
		}).Error("healthcheck failed")
		http.Error(w, http.StatusText(503), 503)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(204)
}

func (e *env) logsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/logplex-1" {
		http.Error(w, http.StatusText(415), 415)
		return
	}
	token := r.Header.Get("Logplex-Drain-Token")
	count, err := strconv.ParseInt(r.Header.Get("Logplex-Msg-Count"), 10, 32)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("unable to parse Logplex-Msg-Count header")
		http.Error(w, http.StatusText(400), 400)
	}

	lines, err := syslog.Scan(r.Body, count)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could not process body")
		http.Error(w, http.StatusText(400), 400)
		return
	}
	_, err = e.db.Insert(token, lines)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could not store logs")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(204)
}

func (e *env) listHandler(w http.ResponseWriter, r *http.Request) {
	token := pat.Param(r, "token")

	logs, err := e.db.List(token)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could find stored logs")
		if err == ds.ErrNoSuchToken {
			http.Error(w, http.StatusText(404), 404)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
	for _, line := range logs {
		w.Write([]byte(line))
	}
}

func (e *env) welcomeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := ioutil.ReadFile("magic_link.key")
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could not read magic link token")
		http.Error(w, http.StatusText(500), 500)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
	w.Write([]byte("Got it: "))
	w.Write(token)
}

func main() {
	listen := os.Getenv("LISTEN")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	keep, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		keep = DefaultBufferSize
	}

	e := &env{}
	switch os.Getenv("DATASTORE") {
	case "redis":
		url, err := url.Parse(os.Getenv("REDIS_URL"))
		if err != nil || url.Scheme != "redis" {
			log.Fatal("$REDIS_URL must be set and valid")
		}
		size, err := strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
		if err != nil {
			size = DefaultRedisPoolSize
		}
		db, err := ds.NewInRedis(url, keep, size)
		if err != nil {
			log.Fatal(err)
		}
		e.db = db
	case "memory":
		fallthrough
	default:
		db, _ := ds.NewInMemory(keep)
		e.db = db
	}

	var (
		root = goji.NewMux()
		list = goji.SubMux()
		logs = goji.SubMux()
	)

	root.HandleFunc(pat.Get("/healthcheck"), e.healthHandler)
	root.Handle(pat.New("/logs"), logs)
	root.Handle(pat.New("/list/*"), list)
	root.HandleFunc(pat.Get("/welcome"), e.welcomeHandler)

	list.HandleFunc(pat.Get("/:token"), e.listHandler)

	logs.Use(auth.DrainTokenAuth(os.Getenv("DRAIN_TOKENS")))
	logs.HandleFunc(pat.Post(""), e.logsHandler)

	if err := http.ListenAndServe(listen+":"+port, root); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("unable to start server")
	}
}
