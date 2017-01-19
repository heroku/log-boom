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
- [x] Log Drain Endpoint
  - [x] Basic Endpoint (No Auth)
  - [x] Heroku Drain Token Auth
- [ ] List Endpoint
  - [x] Basic Endpoint (No Auth)
  - [ ] Authentication
  - [ ] Live Tail Streaming
- [x] Healthcheck Endpoint
  - [x] Ensures backend is functional
- [ ] Welcome [Success URL](https://devcenter.heroku.com/articles/app-json-schema#success_url) Endpoint
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

Name | Default | Description
---- | ------- | -----------
__`BUFFER_SIZE`__ | `1500` | _Optional_, controls the size of the ring buffer in log lines.
__`LISTEN`__ | `0.0.0.0` | _Optional_, controls which interface to listen on.
__`PORT`__ | N/A | _Required_, controls which port to listen on, eg 5000.
__`DATASTORE`__ | `memory` | _Optional_, controls which backend to utilize. Available options are `memory` or `redis`.

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
Redis](https://elements.heroku.com/addons/heroku-redis) or any [3rd Party
Redis Add-on](https://elements.heroku.com/addons).

In order to utlize the `memory` datastore two additional environment variables
that can be customized.

Name | Default | Description
---- | ------- | -----------
__`REDIS_URL`__ | N/A | _Required_, controls which redis to connect to. Automatically set when using a [Heroku Redis](https://elements.heroku.com/addons/heroku-redis).
__`REDIS_POOL_SIZE`__ | `4` |  _Optional_, controls the number of available redis pooled connections.

