## Builder
FROM golang:1.15-buster AS builder

# Copy the code
WORKDIR /go/src/rss-bot
COPY . /go/src/rss-bot

# Build and test
RUN go get -v ./... \
  && go build -v -o /go/build/rss-bot \
  && go test -v ./... 

## Runtime
FROM gcr.io/distroless/base-debian10
COPY --from=builder /go/bin/rss-bot /
ENV BOT_DBPATH /data/bot.db
CMD ["/rss-bot"]
