FROM golang:1.20.3-alpine as builder

WORKDIR /source
COPY . .

RUN go mod download
RUN go build -o chat-server cmd/main.go

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /source/chat-server .
COPY local.env .
COPY prod.env .

CMD [ "./chat-server", "-config-path=prod.env"]