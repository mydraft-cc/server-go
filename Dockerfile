#
# Step 1: Build packages!
#
FROM golang:1.10.1-stretch as builder

# Cache external dependencies
RUN go get cloud.google.com/go/storage \
 && go get github.com/labstack/echo \
 && go get github.com/labstack/echo/middleware \
 && go get github.com/labstack/gommon/log \
 && go get github.com/rs/xid \
 && go get github.com/spf13/viper

WORKDIR /go/src/github.com/athene-wireframes/server

COPY . .

RUN go get && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/server .

#
# Step 2: Build runtime container
#
FROM alpine:latest

RUN apk --no-cache --update add ca-certificates

# Workdir for our app
WORKDIR /app

# Copy from build stage
COPY --from=builder /build/ .

EXPOSE 4000

ENTRYPOINT ["./server"]