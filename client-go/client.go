package client

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	pb "github.com/ItalyPaleAle/rss-bot/model"
)

// Auth token for RPC calls
const authToken = "hello world"

// Timeout for all requests, in seconds
const requestTimeout = 15

// Interval between keepalive requests, in seconds
const keepaliveInterval = 120

// RPCAuth is the object implementing credentials.PerRPCCredentials that provides the auth info
type RPCAuth struct {
	PSK string
}

// GetRequestMetadata returns the metadata containing the authorization key
func (a *RPCAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + a.PSK,
	}, nil
}

// RequireTransportSecurity returns true because this kind of auth requires TLS
func (a *RPCAuth) RequireTransportSecurity() bool {
	return true
}

// BotClient is the client for communicating with the bot (using gRPC)
type BotClient struct {
	actionInfo *pb.ConnectRequest
	routes     map[string]pb.RouteCallback
	client     pb.BotClient
	connection *grpc.ClientConn
	logger     *log.Logger
}

// Init the client
func (c *BotClient) Init(name string, displayName string, helpText string) error {
	// Initialize the logger
	c.logger = log.New(os.Stdout, "grpc: ", log.Ldate|log.Ltime|log.LUTC)

	// Create the actionInfo object
	if name == "" || displayName == "" {
		return errors.New("name and display name must be set")
	}
	c.actionInfo = &pb.ConnectRequest{
		Name:        name,
		DisplayName: displayName,
		HelpText:    helpText,
	}

	// Create the list of routes
	c.routes = make(map[string]pb.RouteCallback)

	return nil
}

// Start the bot and establish a connection with the bot server, then registers the bot
func (c *BotClient) Start() (err error) {
	return c.connect()
}

// starts the connection to the gRPC server and registers the bot
func (c *BotClient) connect() (err error) {
	// Underlying connection
	connOpts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			// Enable InsecureSkipVerify because our certificate is self-signed
			// TODO: Remove this in production!
			InsecureSkipVerify: true,
		})),
		grpc.WithPerRPCCredentials(&RPCAuth{
			PSK: authToken,
		}),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: time.Duration(requestTimeout) * time.Second,
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    time.Duration(keepaliveInterval) * time.Second,
			Timeout: time.Duration(requestTimeout) * time.Second,
		}),
	}
	// TODO: Use env var!
	c.connection, err = grpc.Dial("localhost:2400", connOpts...)
	if err != nil {
		return err
	}

	// Client
	c.client = pb.NewBotClient(c.connection)

	// In another goroutine, make the request to register the action with the bot server, which also creates a stream
	go func() {
		// Continue re-connecting automatically if the connection drops, for as long ast the underlying connection is active
		for c.connection != nil {
			c.logger.Println("Registering the bot")
			// Note that if the underlying connection is down, this call blocks until it comes back
			c.startConnection()
			// Wait 1 second before trying to reconnect
			time.Sleep(1 * time.Second)
		}
	}()

	return nil
}

// Stop closes the connection with the gRPC server
func (c *BotClient) Stop() error {
	conn := c.connection
	c.connection = nil
	err := conn.Close()
	return err
}

// Restart re-connects to the gRPC server
func (c *BotClient) Restart() error {
	if c.connection != nil {
		// Ignore errors here
		_ = c.Stop()
	}
	return c.Start()
}

// startConnection starts the stream with the server
func (c *BotClient) startConnection() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Populate actionInfo with the list of routes
	c.actionInfo.Routes = make([]string, len(c.routes))
	i := 0
	for k := range c.routes {
		c.actionInfo.Routes[i] = k
		i++
	}

	// Connect to the stream RPC
	stream, err := c.client.Connect(ctx, c.actionInfo, grpc.WaitForReady(true))
	if err != nil {
		c.logger.Println("Error while connecting to the bot:", err)
		return
	}
	defer stream.CloseSend()
	c.logger.Println("Bot connected")

	// Watch for incoming messages in a background goroutine
	go func() {
		for {
			// This call is blocking
			in, err := stream.Recv()
			if err == io.EOF {
				c.logger.Println("Stream reached EOF")
				cancel()
				break
			}
			if err != nil {
				c.logger.Println("Error while reading message:", err)
				break
			}

			// Ensure we have a message
			if in == nil || in.Message == nil {
				c.logger.Println("Received empty message")
				continue
			}

			// Ensure that the message can be routed
			if in.ActionName != c.actionInfo.Name {
				c.logger.Println("Received message with invalid action name")
				continue
			}
			if in.ActionRoute == "" {
				c.logger.Println("Received message with empty action route")
				continue
			}

			// Get and invoke the callback
			cb, ok := c.routes[in.ActionRoute]
			if ok && cb != nil {
				cb(in.Message)
				if err != nil {
					// Log errors only
					c.logger.Printf("Callback processing message for chat %d returned an error: %s\n", in.Message.ChatId, err.Error())
					continue
				}
			} else {
				c.logger.Println("Received message for a route that is not available in this bot")
				continue
			}
		}
	}()

	// Wait until the context is canceled (when the connection is closed)
	<-ctx.Done()
	c.logger.Println("Channel closed")
}

// AddRoute adds a route for text messages
// Note that this must be called before invoking the Start method
func (c *BotClient) AddRoute(path string, cb pb.RouteCallback) error {
	if len(path) < 1 {
		return errors.New("route is empty or invalid")
	}
	if cb == nil {
		return errors.New("callback is empty")
	}

	// Add the route to the list
	c.routes[path] = cb

	return nil
}

// SendMessageToRecipient sends a message to a chat or user
func (c *BotClient) SendMessageToRecipient(recipient int64, content interface{}) (*pb.SentMessage, error) {
	// Get the message object
	msg, err := pb.MessageFromContent(content)
	if err != nil {
		return nil, err
	}

	// Set the recipient
	msg.Recipient = strconv.FormatInt(recipient, 10)

	// Send the message
	return c.SendMessage(msg)
}

// SendMessage sends a message (already formatted in a OutMessage object) to a chat or user
func (c *BotClient) SendMessage(msg *pb.OutMessage) (*pb.SentMessage, error) {
	// Send the request to the bot through gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout)*time.Second)
	defer cancel()
	res, err := c.client.SendMessage(ctx, &pb.SendMessageRequest{
		Message: msg,
	}, grpc_retry.WithMax(3))
	if err != nil {
		return nil, err
	}
	return res.SentMessage, nil
}

// EditTextMessage requests an edit to a text message that was sent before
func (c *BotClient) EditTextMessage(sentMsg *pb.SentMessage, text *pb.OutTextMessage, opts *pb.OutMessageOptions) error {
	// Send the request to the bot through gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout)*time.Second)
	defer cancel()
	// Response is empty from this method
	_, err := c.client.EditTextMessage(ctx, &pb.EditTextMessageRequest{
		Message: sentMsg,
		Text:    text,
		Options: opts,
	}, grpc_retry.WithMax(3))
	if err != nil {
		return err
	}
	return nil
}

// RespondToCommand sends a response to a command
// For commands sent in private chats, this just sends a regular message
// In groups, this replies to a specific message
func (c *BotClient) RespondToCommand(in *pb.InMessage, content interface{}) (*pb.SentMessage, error) {
	// Message to send
	msg, err := pb.MessageFromContent(content)
	if err != nil {
		return nil, err
	}

	// Send the request to the bot through gRPC
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestTimeout)*time.Second)
	defer cancel()
	res, err := c.client.RespondToCommand(ctx, &pb.RespondToCommandRequest{
		Message:  in,
		Response: msg,
	}, grpc_retry.WithMax(3))
	if err != nil {
		return nil, err
	}
	return res.SentMessage, nil
}
