FROM golang:1.21
WORKDIR /app
COPY ./ /app/
RUN go build -ldflags "-s -w"
WORKDIR /app
EXPOSE 8080/tcp
ENTRYPOINT ["/app/uts"]
