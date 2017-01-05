# Log Boom

Provides a ring buffer storage for your Heroku logs.

## Work in Progress

This is very much a work in progress project. There are many tasks remaining.
The list below is merely a brain dump of ideas on where to take this project.

- [ ] Configurable Backend Datastores
  - [x] Memory
  - [x] Redis
  - [ ] S3 Bucket
- [x] Multi Drain Capable
- [ ] Log Drain Endpoint
  - [x] Basic Endpoint (No Auth)
  - [] Heroku Drain Token Auth
- [ ] List Endpoint
  - [x] Basic Endpoint (No Auth)
  - [ ] Authentication
  - [ ] Live Tail Streaming
- [x] Healthcheck Endpoint
  - [x] Ensures backend is functional
- [ ] Welcome Endpoint
  - [ ] Guided steps to drain from another app to this collector
  - [ ] Authenticated, perhaps heroku-bouncer style to allow only app collaborators access to guided setup
- [ ] CLI Binary
  - [ ] Dumps `n` items from ring buffer
  - [ ] Dumps `n` items from ring buffer then live tail

## Installation

Click the button

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

## Customization

There are several environment variables that you can tweak to customize your experience

- `BUFFER_SIZE`: Controls the size of the ring buffer. __Optional__, defaults to `1500` log lines.
- `LISTEN`: Controls which interface to listen on. __Optional__, defaults to `0.0.0.0`.
- `PORT`: Controls which port to listen on. __Required__, no default..
- `DATASTORE`: Controls which backend to utilize, `memory` or `redis`. __Optional__, defaults to `memory`.

### Backend Datastores

#### Memory Store

The `memory` store will keep the logs buffered in memory. If the application
restarts or crashes in anyway all the store logs are lost.

There is no additional configuration required for the `memory` datastore.

#### Redis Store

The `redis` datastore will utilize redis to store the logs. The persistance of
the data in that redis datastore is completly up to the owner of that redis
server. 

The simplest way to to wire up a redis datastore is to use [Heroku
Redis](https://elements.heroku.com/addons/heroku-redis) or any 3rd party
[Heroku Redis Add-on](https://elements.heroku.com/addons).

In order to utlize the `memory` datastore two additional environment variables
must be present.

* `REDIS_URL``: Controls which redis to connect to. __Required, but automatically set when using a Heroku Add-on__
* `REDIS_POOL_SIZE`: Controls the number of available redis pooled connections. __Optional__, defaults to `4`.

