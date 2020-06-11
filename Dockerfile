FROM golang:1.14.4 AS builder

WORKDIR /app
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download
COPY ./main.go .
COPY ./pkg ./pkg
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o main .

FROM scratch
COPY --from=builder /app/main /app/
WORKDIR /app
CMD ["/app/main"]