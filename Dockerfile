FROM golang:1.10.1 as build
RUN go get -u golang.org/x/vgo
WORKDIR $GOPATH/src/chat

# Caching dependencies only
COPY go.mod .
COPY cmd/chat/main.go .
RUN vgo verify

# Adding the rest of the code
ADD cmd/chat .
RUN vgo install ./...

# Downloading test deps
RUN vgo test -run=none

FROM gcr.io/distroless/base
COPY --from=build /go/bin/chat /
CMD ["/chat"]