package main

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	ds "github.com/voidlock/log-boom/datastore"
)

// DefaultRedisPoolSize is the default pool size (defaults to 4).
const DefaultRedisPoolSize = 4

const (
	// MinSyslogFrameSize is the smallest (all NULLVALUE) size a syslog frame can be.
	MinSyslogFrameSize = 18

	// MaxSyslogFrameSize is the largest size a syslog frame can be.
	MaxSyslogFrameSize = 10 * 1024
)

type env struct {
	db ds.Datastore
}

func (e *env) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

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
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
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

	lines, err := process(r.Body, count)
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
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	// FIXME handle subtrees properly
	token := r.URL.Path

	logs, err := e.db.List(token)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could not store logs")
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

// ScanRFC6587 does stuff
func ScanRFC6587(data []byte, atEOF bool) (int, []byte, error) {
	mark := 0
	for ; mark < len(data); mark++ {
		if data[mark] == ' ' {
			break
		}
	}

	for i := mark; i < len(data); i++ {
		if data[i] == '<' {
			offset, err := strconv.Atoi(string(data[0:mark]))
			if err != nil {
				return 0, nil, err
			}
			token := data[:mark+offset+1]
			return len(token), token, nil
		}
	}

	if atEOF && len(data) > mark {
		return 0, nil, errors.New("Not RFC6587 Formatted Syslog")
	}

	// Request more data.
	return mark, nil, nil
}

func process(r io.Reader, count int64) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Split(ScanRFC6587)

	lines := make([]string, 0, count)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func main() {
	listen := os.Getenv("LISTEN")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	keep, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		keep = 1500
	}

	e := &env{}
	switch os.Getenv("DATASTORE") {
	default:
		db, _ := ds.NewInMemory(keep)
		e.db = db
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
	}

	http.HandleFunc("/healthcheck", e.healthHandler)
	http.HandleFunc("/logs", e.logsHandler)
	http.Handle("/list/", http.StripPrefix("/list/", http.HandlerFunc(e.listHandler)))

	if err := http.ListenAndServe(listen+":"+port, nil); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("unable to start server")
	}
}
