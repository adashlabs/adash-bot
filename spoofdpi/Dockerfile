FROM golang:alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
RUN git clone https://github.com/xvzc/SpoofDPI.git .
RUN go build -o spoofdpi cmd/spoofdpi/main.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/spoofdpi /usr/local/bin/spoofdpi
EXPOSE 8080
ENTRYPOINT ["spoofdpi"]
