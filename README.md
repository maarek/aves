# AvES

[![GoVer](https://img.shields.io/github/go-mod/go-version/maarek/aves.svg)](https://github.com/maarek/aves/blob/master/go.mod)
[![GoDoc](https://godoc.org/github.com/maarek/aves?status.svg)](https://godoc.org/github.com/maarek/aves)
[![CircleCI](https://circleci.com/gh/maarek/aves/tree/master.svg?style=svg)](https://circleci.com/gh/maarek/aves/tree/master)
[![Release](https://github.com/maarek/aves/workflows/release/badge.svg)](https://github.com/maarek/aves/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/maarek/aves)](https://goreportcard.com/report/github.com/maarek/aves)
[![Test Coverage](https://api.codeclimate.com/v1/badges/7cea74edf35eb31b5427/test_coverage)](https://codeclimate.com/github/maarek/aves/test_coverage)
[![Maintainability](https://api.codeclimate.com/v1/badges/7cea74edf35eb31b5427/maintainability)](https://codeclimate.com/github/maarek/aves/maintainability)


## Introduction

AvES is an that is tailored towards storing Events for _Event Sourcing_.

It uses the _RESP_ (REdis Serialization Protocol) to communicate.
This way it is possible to create clients by reusing the already available protocol. For example, it is possible to use [the official Redis command line interface](https://redis.io/topics/rediscli) program to communicate with aves.

An event store is a datastore that stores events on disk indefinitely. An initial event sent for an aggregate stream creates the store of all events for that stream. 

## Features

- Event publication
- Event stream subscription
- Event stream fetch over range
- Redis based protocol

## Building aves

```bash
git clone https://github.com/maarek/aves.git
cd aves
make build
```

## Basic Event Store Usage

Once AvES is installed and available in your `PATH`, you can run it by executing the following command.

```bash
aves --out mydb.aves
```

There is now a aves server running on your machine and listening on `127.0.0.1:6480`.
In another terminal window, you can specify to a client to listen to only new events.

```bash
avcli "SUBSCRIBE" 'my-stream'
```

And in another one again you can send new events.
All clients which are subscribed to that same stream will see the events.

```bash
avcli publish 'my-stream' '1' 'Hello World!'
avcli publish 'my-stream' '2' 'Hello America!'
avcli publish 'my-stream' '3' 'Hello Africa!'
avcli publish 'my-stream' '4' 'Hello Japan!'
```
