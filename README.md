# Agent

Agent is part of the Generic Container Orchestration project.
It is the most crucial part of the project as it is responsible for managing the state
of the external container systems.

The agent keeps the desired state and the actual state in sync in both directions:

- Changes applied through the gRPC/HTTP API are reconciled towards the container provider.
- The agent subscribes to the provider's event stream (e.g. the docker events API) and
  periodically resyncs, so containers that crash or are removed outside of the agent are
  automatically restored to the desired state (self-healing).

## Usage

Requires Go 1.25+ and a running docker daemon.

If you are using docker desktop, make sure you have allowed the default docker socket to be used option under the advanced tab in the settings.

### Install dependencies
```bash
# Install packages
go get .
```

### Start app
```bash
# Start app, gRPC and http server
make run
```

### Using the API
> [!TIP]  
> You can use our postman collection with all the pre-made api calls made for you.  
> You can find the postman collection [here](https://www.postman.com/galactic-spaceship-310683/workspace/gco/collection/3303581-894b7592-c8d1-47c7-8a14-d46dd88af130?action=share&creator=3303581).  
> While executing the api requests keep a close look on your docker container you have created.

The following commands can be used to perform required actions upon the code base.

```bash
# run all tests
make test
```

## Application specification

An application is described with the following fields (JSON shown for the HTTP API,
the same shape applies to the gRPC API):

```json
{
  "application": {
    "name": "my-app",
    "image": { "name": "nginx", "tag": "alpine", "pull_policy": "always" },
    "ports": [{ "container_port": 80, "host_port": 8090, "protocol": 1 }],
    "instances": 1,
    "env": { "APP_MODE": "production" },
    "volumes": [{ "source": "my-volume", "destination": "/data", "read_only": false }],
    "labels": { "team": "platform" },
    "network_mode": "bridge",
    "restart_policy": "unless-stopped",
    "health_check": {
      "test": ["CMD", "curl", "-f", "http://localhost/"],
      "interval_seconds": 10,
      "timeout_seconds": 3,
      "retries": 3,
      "start_period_seconds": 5
    },
    "resources": { "memory": "256m", "cpus": 0.5 }
  }
}
```

- `env`, `volumes`, `labels`, `network_mode`, `restart_policy`, `health_check` and
  `resources` are optional. Both `snake_case` and `camelCase` field names are accepted.
- `labels` may not use the reserved `gco.io` namespace.
- `restart_policy` accepts `no`, `always`, `on-failure` or `unless-stopped`. While a
  docker native restart policy is restarting a container, the agent leaves it alone.
- `health_check.test` must start with `CMD`, `CMD-SHELL` or `NONE`.
- `resources.memory` accepts human readable values such as `512m` or `1g`, with a
  minimum of `6m` (the docker daemon minimum).
- A volume `source` is either an absolute host path or the name of a docker volume.

## Supported Providers

| Provider     | Description                                                                                                                                    | Version   |
|--------------|------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| Docker       | The docker provider lets you manage a single system using docker. It will communicate with the local docker socket and apply changes as needed | `v0.1.0+` |
| Docker Swarm | N/A                                                                                                                                            | N/A       |
| Kubernetes   | N/A                                                                                                                                            | N/A       |

## License

[GNU](./LICENSE)
