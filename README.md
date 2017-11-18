# Centrifugo exporter [![Build Status](https://travis-ci.org/nordicdyno/centrifugo_exporter.svg)][travis]

[![Docker build](https://img.shields.io/docker/build/nordicdyno/centrifugo_exporter.svg)][hub]
[![Docker Pulls](https://img.shields.io/docker/pulls/nordicdyno/centrifugo_exporter.svg)][hub]

prometheus exporter for [Centrifugo](https://github.com/centrifugal/centrifugo) server

## how to run and test

with existing Centrifugo instance:

    docker run -p 9315:9315 nordicdyno/centrifugo_exporter -centrifugo.server=CENTRIFUGO_ADDRESS -centrifugo.secret=CENTRIFUGO_SECRET -centrifugo.timeout=1s

just testing with Centrifugo in docker:

    docker-compose up
    curl http://localhost:9315/metrics

## how it works

scrapes `/node` API endpoint and exports scraped metrics in Prometheus format

[`/node` method documentation](https://fzambia.gitbooks.io/centrifugal/content/server/api.html#node-centrifugo--140)

## how to install

    go install github.com/nordicdyno/centrifugo_exporter

## how to use

    centrifugo_exporter -centrifugo.secret=<cent-secret-here> -centrifugo.server=<cent-addr-here> -centrifugo.timeout=500ms

## metrics supported

Supports most important metrics for now, ignores aggregates like percentiles calculated by centrifugo.

Metrics list:

* `up` - gauge states is current scrape successful or not
* `client_bytes_in_total` - client API inbound traffic (bytes sent from clients)
* `client_bytes_out_total` - client API outbound traffic (bytes sent to clients)
* `client_num_connectz` - connections of client API
* `client_num_msg_published` - messages published via client API
* `client_num_msg_queued` - messages put into client queues
* `client_num_msg_sent`- messages actually sent to client
* `client_num_subscribe` - subscribes via client API
* `node_num_clients` - current number of connected authorized clients
* `node_num_unique_clients` - current number of unique clients connected
* `node_num_channels` - current number of active channels
* `node_num_client_msg_published` -  messages published
* `http_api_num_requests` - requests to server HTTP API

## TODO

* more counters and gauges (probably to add all of them)
* export percentiles
* binaries on releases page

[travis]: https://travis-ci.org/nordicdyno/centrifugo_exporter
[hub]: https://hub.docker.com/r/nordicdyno/centrifugo_exporter/
