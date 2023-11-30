# agent

Agent is part of the Generic Container Orchestration project.
It is the most crucial part of the project as it is responsible for managing the state
of the external container systems.

## Usage

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
> You can use our postman collection with all the api calls made for you.
> You can find the postman collection [here](https://www.postman.com/galactic-spaceship-310683/workspace/gco/collection/3303581-894b7592-c8d1-47c7-8a14-d46dd88af130?action=share&creator=3303581)
> While executing the api requests keep a close look on your docker container you have created

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
