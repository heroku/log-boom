package datastore

import "errors"

// Errors returned by Datastore.
var (
	ErrNoSuchToken = errors.New("no such token")
)

// Datastore is the main interface into the db package.
type Datastore interface {
	HealthChecker
	Inserter
	Lister
}

// Inserter is the interface for inserting records into the Datastore.
type Inserter interface {
	Insert(token string, lines []string) (int, error)
}

// HealthChecker is the interface for performming healthchecks against the Datastore.
type HealthChecker interface {
	Healthcheck() (bool, error)
}

// Lister is the interface for listing logs stored in the Datastore.
type Lister interface {
	List(token string) ([]string, error)
}
