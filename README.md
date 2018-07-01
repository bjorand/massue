# massue

Massue is a load testing tool written in Golang.

## Usage

```
massue -h
Usage of massue:
  -c int
    	Number of multiple requests to make at a time (default 1)
  -n int
    	Number of requests to perform (default 1)
  -u string
    	URL
```

## Visualization

Massue can push metrics to statsd.
The following documentation helps you setting up Grafana, Graphite and Stats with docker containers:
```
git clone git@github.com:kamon-io/docker-grafana-graphite.git
cd docker-grafana-graphite
make up
```
Visit http://localhost to connect to Grafana. Credentials are admin/admin.
