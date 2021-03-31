package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/ItalyPaleAle/rss-bot/core/bot"
	pb "github.com/ItalyPaleAle/rss-bot/model"
)

// RPCServer manages the gRPC server
type RPCServer struct {
	pb.UnimplementedBotServer

	Ctx           context.Context
	bot           *bot.BotManager
	providers     []string
	log           *log.Logger
	stopCh        chan int
	restartCh     chan int
	doneCh        chan int
	runningCtx    context.Context
	runningCancel context.CancelFunc
	running       bool
	grpcServer    *grpc.Server
}

// Init the gRPC server
func (s *RPCServer) Init(bot *bot.BotManager) {
	s.running = false

	// If there's no context, use the background one
	if s.Ctx == nil {
		s.Ctx = context.Background()
	}

	// Store the bot manager
	s.bot = bot

	// Providers slice
	s.providers = []string{}

	// Initialize the logger
	s.log = log.New(os.Stdout, "server: ", log.Ldate|log.Ltime|log.LUTC)

	// Channels used to stop and restart the server
	s.stopCh = make(chan int)
	s.restartCh = make(chan int)
	s.doneCh = make(chan int)
}

// Start the gRPC server
func (s *RPCServer) Start() {
	for {
		// Create the context
		s.runningCtx, s.runningCancel = context.WithCancel(s.Ctx)

		// TLS
		// TODO: MAKE CERTIFICATE LOCATION CONFIGURABLE
		creds, err := credentials.NewServerTLSFromFile("cert.pem", "key.pem")
		if err != nil {
			panic(err)
		}

		// Create the server
		s.grpcServer = grpc.NewServer(
			grpc.Creds(creds),
			grpc.UnaryInterceptor(authUnaryInterceptor),
			grpc.StreamInterceptor(authStreamInterceptor),
		)
		pb.RegisterBotServer(s.grpcServer, s)

		// Start the server in another channel
		go func() {
			// Listen
			// TODO: Use a variable to set the port and address
			port := 2400
			listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err != nil {
				s.runningCancel()
				panic(err)
			}
			s.log.Printf("Starting gRPC server on port %d\n", port)
			s.running = true
			s.grpcServer.Serve(listener)
		}()

		select {
		case <-s.stopCh:
			// We received a signal to stop the server; shut down for good
			s.log.Println("Shutting down the gRCP server")
			s.gracefulStop()
			s.running = false
			s.doneCh <- 1
			return
		case <-s.restartCh:
			// We received a signal to restart the server
			s.log.Println("Restarting the gRCP server")
			s.gracefulStop()
			s.doneCh <- 1
			// Do not return, let the for loop repeat
		}
	}
}

// Restart the server
func (s *RPCServer) Restart() {
	if s.running {
		s.restartCh <- 1
		<-s.doneCh
	}
}

// Stop the server
func (s *RPCServer) Stop() {
	if s.running {
		s.stopCh <- 1
		<-s.doneCh
	}
}

// Internal function that gracefully stops the gRPC server, with a timeout
func (s *RPCServer) gracefulStop() {
	const shutdownTimeout = 15
	ctx, cancel := context.WithTimeout(s.Ctx, time.Duration(shutdownTimeout)*time.Second)
	defer cancel()

	// Cancel the context
	s.runningCancel()

	// Try gracefulling closing the gRPC server
	closed := make(chan int)
	go func() {
		s.grpcServer.GracefulStop()
		if closed != nil {
			// Use a select just in case the channel was closed (which would cause a panic)
			select {
			case closed <- 1:
			default:
			}
		}
	}()

	select {
	// Closed - all good
	case <-closed:
		close(closed)
	// Timeout
	case <-ctx.Done():
		// Force close
		s.log.Printf("Shutdown timeout of %d seconds reached - force shutdown\n", shutdownTimeout)
		s.grpcServer.Stop()
		close(closed)
		closed = nil
	}
	s.log.Println("gRPC server shut down")
}

// Connect is the handler for the Connect gRPC
func (s *RPCServer) Connect(req *pb.ConnectRequest, stream pb.Bot_ConnectServer) (err error) {
	// Ensure that the required fields are set: name and display name
	if req.Name == "" || req.DisplayName == "" {
		return status.Error(codes.InvalidArgument, "name and display name must be set")
	}

	// Ensure that no provider with the same name is already registered
	for _, e := range s.providers {
		if e == req.Name {
			return status.Error(codes.AlreadyExists, "a provider with the same name already exists")
		}
	}

	// Add all routes (if any)
	if len(req.Routes) > 0 {
		// Callback that forwards all incoming messages to the action provider
		cb := func(action string, route string) func(msg *pb.InMessage) error {
			return func(msg *pb.InMessage) error {
				return stream.Send(&pb.MessagesStream{
					ActionName:  action,
					ActionRoute: route,
					Message:     msg,
				})
			}
		}

		// Add all routes
		for _, route := range req.Routes {
			err = s.bot.AddRoute(req.Name, route, cb(req.Name, route))
			if err != nil {
				return status.Error(codes.Internal, err.Error())
			}
		}
	}

	// TODO: ADD HELP TEXT

	s.log.Printf("Provider %s (%s) connected\n", req.Name, req.DisplayName)

	// Maintain this goroutine running for as long as the connection is open
	select {
	// Exit if context is done (i.e. connection is closed)
	case <-stream.Context().Done():
		break

	// The server is shutting down
	case <-s.runningCtx.Done():
		break
	}

	// Unregister the provider
	s.bot.RemoveProvider(req.Name)
	s.log.Printf("Provider %s disconnected\n", req.Name)

	return nil
}

// SendMessage is the handler for the SendMessage gRPC
func (s *RPCServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (res *pb.SendMessageResponse, err error) {
	// Send the message to the bot
	out, err := s.bot.SendMessage(req.Message)
	if err != nil {
		return &pb.SendMessageResponse{}, status.Error(codes.Internal, err.Error())
	}

	// Respond with the message that was sent
	return &pb.SendMessageResponse{
		SentMessage: out,
	}, nil
}

// RespondToCommand is the handler for the RespondToCommand gRPC
func (s *RPCServer) RespondToCommand(ctx context.Context, req *pb.RespondToCommandRequest) (res *pb.RespondToCommandResponse, err error) {
	// Send the request to the bot
	out, err := s.bot.RespondToCommand(req.Message, req.Response)
	if err != nil {
		return &pb.RespondToCommandResponse{}, status.Error(codes.Internal, err.Error())
	}

	// Respond with the message that was sent
	return &pb.RespondToCommandResponse{
		SentMessage: out,
	}, nil
}

// EditTextMessage is the handler for the EditTextMessage gRPC
func (s *RPCServer) EditTextMessage(ctx context.Context, req *pb.EditTextMessageRequest) (res *pb.EditTextMessageResponse, err error) {
	// Send the request to the bot
	err = s.bot.EditTextMessage(req.Message, req.Text, req.Options)
	if err != nil {
		return &pb.EditTextMessageResponse{}, status.Error(codes.Internal, err.Error())
	}

	// Currently there's no response
	return &pb.EditTextMessageResponse{}, nil
}

// Interceptor for unary ("simple RPC") requests that checks the authorization metadata
func authUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// Check if the call is authorized
	err = checkAuth(ctx)
	if err != nil {
		return
	}

	// Call is authorized, so continue the execution
	return handler(ctx, req)
}

// Interceptor for stream requests that checks the authorization metadata
func authStreamInterceptor(srv interface{}, srvStream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	// Check if the call is authorized
	err = checkAuth(srvStream.Context())
	if err != nil {
		return
	}

	// Call is authorized, so continue the execution
	return handler(srv, srvStream)
}

// Used by the interceptors, this checks the authorization metadata
func checkAuth(ctx context.Context) error {
	// Ensure we have an authorization metadata
	// Note that the keys in the metadata object are always lowercased
	m, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "missing authorization metadata")
	}
	if len(m["authorization"]) != 1 {
		return status.Error(codes.Unauthenticated, "invalid authorization")
	}

	// Remove the optional "Bearer " prefix
	if strings.TrimPrefix(m["authorization"][0], "Bearer ") != "hello world" {
		return status.Error(codes.Unauthenticated, "invalid authorization")
	}

	// All good
	return nil
}
