#### Docker

Follow these steps to compile the project, create a docker image and setting up a container:

1. To install the Docker Compose plugin within the Docker server, use the following command..
```
% yum install docker-compose-plugin
```

2. Compile the project and create a new image. Start the container with the generated image..
```
% docker build -t cvtunnel:latest .
```

```
docker run -d --name cloudvalley_tunnel \
  --network cloudvalley_network \
  --restart unless-stopped \
  :latest \
  server --port 8000 --reverse --auth cloudvalley:developer910
```




