# Introduction

This is a live tutorial given at LinuxCon 2016 North America. It is designed to introduce all of the basic concepts of CoreOS and Kubernetes as a platform for application management and distributed systems.

Slides: https://speakerdeck.com/philips/coreos-a-tutorial-on-hyperscale-infrastructure

## Building Guestbook

Releasing the image requires that you have access to the registry user account which will host the image. You can specify the registry including the user account by setting the environment variable `REGISTRY`.

To build and release the guestbook image:

    cd examples/guestbook-go/_src
    make release

To build and release the guestbook image with a different registry and version:

    VERSION=v4 REGISTRY="quay.io/philips" make build

If you want to, you can build and push the image step by step:

    make clean
    make build
    make push


## Developing

Port forward redis to localhost

```
kubectl port-forward $(kubectl get pods -l app=redis,role=master -o template --template="{{range.items}}{{.metadata.name}}{{end}}") 6380:6379
kubectl port-forward $(kubectl get pods -l app=redis,role=slave -o template --template="{{range.items}}{{.metadata.name}}{{end}}") 6379:6379
```

Tell the app to use those ports

```
REDIS_SLAVE=localhost:6379 REDIS_MASTER=localhost:6380 go run main.go
```
