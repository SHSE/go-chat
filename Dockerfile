FROM golang:1.10.3 as build
RUN go get -u golang.org/x/vgo
WORKDIR $GOPATH/src/chat
ADD . ./
RUN vgo install ./...

# Downloading test deps
RUN vgo test -run=none ./chat ./transport

FROM gcr.io/distroless/base
COPY --from=build /go/bin/chat /
CMD ["/chat"]