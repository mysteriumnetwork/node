FROM frolvlad/alpine-glibc

RUN apk add --no-cache curl

RUN mkdir -p /ethereum

WORKDIR "/ethereum"
VOLUME "/ethereum/keystore"
VOLUME "/ethereum/geth-runtime"

RUN curl https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.8.12-37685930.tar.gz \
    | tar --strip 1 -xvz

ENTRYPOINT ["./geth","--datadir","geth-runtime","--keystore","keystore"]
