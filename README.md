# Centrifugo exporter

prometheus exporter for [centrifugo](https://github.com/centrifugal/centrifugo) server

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

* `up` - is scrape succesful or not
* `client_bytes_in_total` - number of bytes coming to client API (bytes sent from clients)
* `client_bytes_out_total` - number of bytes coming out of client API (bytes sent to clients)
* `client_num_connectz` - number of connections of client API
* `client_num_msg_published` - number of messages published via client API
* `client_num_msg_queued` - number of messages put into client queues
* `client_num_msg_sent`- number of messages actually sent to client
* `client_num_subscribe` - subscribes via client API
* `node_num_clients` - number of connected authorized clients
* `node_num_unique_clients` - number of unique clients connected
* `node_num_channels` - number of active channels
* `node_num_client_msg_published` - number of messages published
* `http_api_num_requests` - number of requests to server HTTP API

All metrics are exported as gauges for simplicity (should change in near future)

## TODO

* use counters in exported metrics for actual counters
* binaries on releases page
* docker support
