#
# Step 1: Build packages!
#
FROM golang:1.17.3-bullseye as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /build

#
# Step 2: Build runtime container
#
FROM gcr.io/distroless/base-debian10

# Workdir for our app
WORKDIR /

# Copy from build stage
COPY --from=builder /build/ /build

EXPOSE 4000

USER nonroot:nonroot

ENTRYPOINT ["/build"]