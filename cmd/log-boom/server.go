package main

import (
	"bufio"
	"errors"
	"fmt"
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
	stored, err := e.db.Insert(token, lines)
	if err != nil {
		log.WithFields(log.Fields{
			"at":  "logs",
			"err": err,
		}).Error("could not store logs")
		http.Error(w, http.StatusText(500), 500)
		return
	}

	log.WithFields(log.Fields{
		"at":     "logs",
		"len":    len(lines),
		"stored": stored,
	}).Info("stored logs in redis")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(204)
}

const (
	// MinSyslogFrameSize is the smallest (all NULLVALUE) size a syslog frame can be.
	MinSyslogFrameSize = 18

	// MaxSyslogFrameSize is the largest size a syslog frame can be.
	MaxSyslogFrameSize = 10 * 1024
)

func scanOffset(data []byte, atEOF bool) (int, error) {
	advance, token, err := bufio.ScanWords(data, atEOF)
	if token == nil || err != nil {
		return 0, err
	}

	offset, err := strconv.ParseInt(string(token), 10, 32)
	if err != nil {
		return 0, err
	}
	return advance + int(offset), nil
}

func readSyslogFrame(data []byte, atEOF bool) (int, []byte, error) {

	advance, err := scanOffset(data, atEOF)
	if err != nil {
		return 0, nil, err
	}

	if advance < MinSyslogFrameSize || advance > MaxSyslogFrameSize {
		return 0, nil, errors.New("Invalid Syslog Frame")
	}

	if !atEOF && advance > len(data) {
		return 0, nil, nil
	}

	if atEOF && len(data)-advance < MinSyslogFrameSize {
		return advance, data[:advance], bufio.ErrFinalToken
	}

	return advance, data[:advance], nil
}

func process(r io.Reader, count int64) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Split(readSyslogFrame)

	lines := make([]string, 0, count)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		fmt.Printf("%#v\n", lines)
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
	url, err := url.Parse(os.Getenv("REDIS_URL"))
	if err != nil || url.Scheme != "redis" {
		log.Fatal("$REDIS_URL must be set and valid")
	}
	keep, err := strconv.Atoi(os.Getenv("BUFFER_SIZE"))
	if err != nil {
		keep = 1500
	}
	size, err := strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	if err != nil {
		size = DefaultRedisPoolSize
	}

	db, err := ds.NewRedisBackend(url, keep, size)
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
