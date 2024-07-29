FROM golang:1.22-bullseye as base

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/app" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 65532 \
    small-user

WORKDIR /app

COPY go.mod go.sum /app/
RUN go mod download
RUN go mod verify

FROM scratch
COPY --from=base /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group

FROM base
COPY main.go .
COPY templates ./templates

ARG GIT_SHA=unknown
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.gitSHA=$GIT_SHA" -o uts

WORKDIR /app
COPY --from=base /app/uts /app/uts
COPY --from=base /app/templates/* /app/templates/

USER small-user:small-user
EXPOSE 8080/tcp
ENTRYPOINT ["/app/uts"]
