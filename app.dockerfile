FROM golang:1.21 as builder
WORKDIR /build
COPY go.mod . 
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ./writer ./cmd/test_writer 
RUN CGO_ENABLED=0 GOOS=linux go build -o ./xapp ./cmd/main 

FROM debian:11
WORKDIR /app
COPY --from=builder ./build/xapp ./xapp
COPY --from=builder ./build/writer ./writer

RUN apt update && apt install -y less 
RUN apt install -y procps 
LABEL autor=ias
ENTRYPOINT ["/app/writer"]