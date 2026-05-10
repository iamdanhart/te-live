ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -tags production -o /run-app .


FROM debian:bookworm

RUN useradd -r -s /bin/false appuser
COPY --from=builder /run-app /usr/local/bin/
USER appuser
CMD ["run-app"]
