# agent

Agent is part of the Generic Container Orchestration project.
It is the most crucial part of the project as it is responsible for managing the state
of the external container systems.

## Usage

The following commands can be used to perform required actions upon the code base.

```bash
# run all tests
make test
```

## Supported Providers

| Provider     | Description                                                                                                                                    | Version   |
|--------------|------------------------------------------------------------------------------------------------------------------------------------------------|-----------|
| Docker       | The docker provider lets you manage a single system using docker. It will communicate with the local docker socket and apply changes as needed | `v0.1.0+` |
| Docker Swarm | N/A                                                                                                                                            | N/A       |
| Kubernetes   | N/A                                                                                                                                            | N/A       |

## License

[GNU](./LICENSE)
