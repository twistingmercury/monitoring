# Observability middleware for gin-gonic/gin: Examples

This repository contains an example of using the packages [monitoring-traces](https://github.com/twistingmercury/monitoring-traces),
[monitoring-logs](https://github.com/twistingmercury/monitoring-logs), and 
[monitoring-metrics](https://github.com/twistingmercury/monitoring-metrics).

## Prerequisites

* Go 1.21.x
* Docker and Docker Compose
* Python v3.x

## Running the example

Simply run `make run`.

A python script is provided so that the endpoint is called continuouly to exercise the example service here:[testclient.py](testclient/client.py).