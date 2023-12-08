FROM golang:1.18-alpine3.17 AS build

ENV CGO_ENABLED=0
ENV APP_DIR=$GOPATH/src/go-server/
RUN mkdir -p $APP_DIR
WORKDIR $APP_DIR

COPY go.mod go.sum $APP_DIR
RUN go mod download

COPY ./ $APP_DIR
RUN go build -o /external-dns-adguard

FROM alpine:3.19 AS prod
COPY --from=build /external-dns-adguard /
ENTRYPOINT /external-dns-adguard
