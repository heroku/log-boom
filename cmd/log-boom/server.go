package main

import (
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func logsHander(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func main() {
	listen := os.Getenv("LISTEN")
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	http.HandleFunc("/health", healthHander)
	http.HandleFunc("/logs", logsHander)

	if err := http.ListenAndServe(listen+":"+port, nil); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Fatal("unable to start server")
	}
}
