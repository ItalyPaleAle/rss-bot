## Builder
FROM golang:1.19-buster AS builder

# Copy the code
WORKDIR /src
COPY . /src

# Build and test
ENV CGO_ENABLED=1
RUN go get -v ./...
RUN mkdir /dist
RUN go build -v -o /dist/rss-bot
RUN go test -v ./... 

## Runtime
FROM gcr.io/distroless/base-debian11:nonroot
COPY --from=builder /dist/rss-bot /
ENV BOT_DBPATH /data/bot.db
CMD ["/rss-bot"]
