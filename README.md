# immutable-storage

## run as Docker container

```
# create docker image
docker build -t immutable-storage .

# start docker container
docker container run -d --network host --name immutable-storage <image id>
```


## health check
```
grpcurl --plaintext 127.0.0.1:3100 server.ImmutableStorage.AvailableKeys
```

### Store message

```
grpcurl --plaintext -d '{"Key": "<Key>", "Content": "<Message>"}' <Source IP>:3100 server.ImmutableStorage.Store
```
