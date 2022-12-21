FROM golang:1.18 AS dev_img

RUN apt update && apt install git

ENV APP_DIR=$GOPATH/src/go-server/
RUN mkdir -p $APP_DIR
WORKDIR $APP_DIR

COPY go.mod go.sum $APP_DIR
RUN go mod download

COPY ./ $APP_DIR
RUN CGO_ENABLED=1 GOOS=linux \
   go build -gcflags "all=-N -l" -o /service

FROM debian:buster-slim AS prod_img
COPY --from=dev_img /service /
ENTRYPOINT /service
