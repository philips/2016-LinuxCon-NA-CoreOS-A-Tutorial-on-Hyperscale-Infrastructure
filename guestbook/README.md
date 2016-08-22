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
