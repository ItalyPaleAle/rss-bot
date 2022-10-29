## Builder
FROM golang:1.19-buster AS builder

# Copy the code
WORKDIR /go/src/rss-bot
COPY . /go/src/rss-bot

# Build and test
ENV CGO_ENABLED=1
RUN go get -v ./... \
  && go build -v -o /go/build/rss-bot \
  && go test -v ./... 

## Runtime
FROM gcr.io/distroless/base-debian11:nonroot
COPY --from=builder /go/bin/rss-bot /
ENV BOT_DBPATH /data/bot.db
CMD ["/rss-bot"]
