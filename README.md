# Iotflow

Iotflow is a proxy with various features that allow to monitor the data flow between downstream and upsteam.

## Features:

- Environment variables configuration;
- Load balancing - (`round-robin`, `least-conn` and `random`);
- Machine learning sentiment check for requests;
- SMS Alerts (via `Infobip` API);
- `Prometheus` metrics collection;
- `Grafana` graphs via `Prometheus`;
- Tools for mocking upstream servers and downstream clients;
- `Docker` setup for `Prometheus` and `Grafana`;

## Prerequisites:

- `Docker`;
- Go 1.17 or later;
- If using linux and generator tool, the `/usr/share/dict/words` file is needed, it can be installed via words package;

## Setup

To launch the `Docker` services, at the root level of the files do:
```
docker-compose up -d
```

Navigate to `/cmd`, copy env.example to env, populate it and do:
```
make
```

To launch mock upstream servers navigate to `/tools/target` and do:
```
go run ./... -port=13308
```

To launch mock downstream clients navigate to `/tools/generator` and do:
```
go run ./...
```

## Post Setup

- Must configure `Grafana` at `127.0.0.1:3000`
- Must configure `Grafana` data source. `Prometheus` port is `9090`.
- Create a new dashboard and graphs.
