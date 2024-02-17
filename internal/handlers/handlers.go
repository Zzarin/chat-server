package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	rpc "github.com/Zzarin/chat-server/pkg/chat_server_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserHandler struct {
	done chan os.Signal

	rpc.UnimplementedChatServerV1Server
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		done: make(chan os.Signal),
	}
}

func (u *UserHandler) ListenAndServe(ctx context.Context, address string) error {
	serverOptions := []grpc.ServerOption{
		// grpc.UnaryInterceptor(), // add interceptor later
		// grpc.StreamInterceptor(), // add interceptor later
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:              30 * time.Second, // Time between pings
			Timeout:           5 * time.Second,  // Timeout for connection to be considered dead
			MaxConnectionIdle: 40 * time.Second, // If a client is idle for 40 seconds, send a GOAWAY
		}),

		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // Minimum time between pings
			PermitWithoutStream: true,             // Allow pings even if no active streams
		}),
	}

	s := grpc.NewServer(serverOptions...)
	reflection.Register(s)
	rpc.RegisterChatServerV1Server(s, u)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	log.Printf("listening for connections on %s", address)

	go func() {
		if err = s.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	s.GracefulStop()
	log.Println("stopped listening for clients...", ctx.Err())
	return nil
}

func (u *UserHandler) Stop() {
	u.done <- os.Interrupt
}

func (u *UserHandler) Create(ctx context.Context, req *rpc.CreateRequest) (*rpc.CreateResponse, error) {
	userNames := req.GetUsernames()
	fmt.Println(userNames)
	return &rpc.CreateResponse{}, nil
}

func (u *UserHandler) SendMessage(ctx context.Context, req *rpc.SendMsgRequest) (*emptypb.Empty, error) {
	from := req.GetFrom()
	text := req.GetText()
	timestamp := req.GetTimestamp()
	fmt.Printf("from: %s\ntext: %s\ntimestamp: %s\n", from, text, timestamp)
	return &emptypb.Empty{}, nil
}

func (u *UserHandler) Delete(ctx context.Context, req *rpc.DeleteRequest) (*emptypb.Empty, error) {
	chatID := req.GetChatId()
	fmt.Println(chatID)
	return &emptypb.Empty{}, nil
}
