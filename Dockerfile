FROM golang:1.9.2
ARG VERSION
WORKDIR /go/src/github.com/nordicdyno/centrifugo_exporter
COPY . .
RUN echo "RUN go build..."
RUN CGO_ENABLED=0 go build -ldflags "-X 'main.VERSION=${VERSION}'" -o centrifugo_exporter .

FROM alpine:latest
RUN echo "build alpine based image..."
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/nordicdyno/centrifugo_exporter/centrifugo_exporter .
CMD ["./centrifugo_exporter"]
