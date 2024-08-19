package handlers

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	rpc "github.com/Zzarin/chat-server/pkg/chat_server_v1"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var dbTimeOutDefault = time.Duration(5 * time.Second)

type UserHandler struct {
	done   chan os.Signal
	dbConn *pgxpool.Pool

	rpc.UnimplementedChatServerV1Server
}

func NewUserHandler(conn *pgxpool.Pool) *UserHandler {
	return &UserHandler{
		done:   make(chan os.Signal),
		dbConn: conn,
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

	builderInsert := sq.Insert("chats").
		PlaceholderFormat(sq.Dollar).
		Columns("users").
		Values(pq.Array(userNames)).
		Suffix("RETURNING id")

	query, args, err := builderInsert.ToSql()
	if err != nil {
		log.Printf("userNames: %v, %v", userNames, errors.Wrap(err, "ToSql"))
		return nil, status.Error(codes.Internal, "preparing query")
	}

	var chatID int
	ctxDB, cancel := context.WithTimeout(ctx, dbTimeOutDefault)
	defer cancel()

	err = u.dbConn.QueryRow(ctxDB, query, args...).Scan(&chatID)
	if err != nil {
		log.Printf("userNames: %v, %v", userNames, errors.Wrap(err, "QueryRow"))
		return nil, status.Error(codes.Internal, "writing in db")
	}

	return &rpc.CreateResponse{ChatId: int64(chatID)}, nil
}

type Message struct {
	chatID int64
	sender string
	text   string
}

func getMessageFromSendMessageRequest(chatID int64, sender, message string) Message {
	return Message{
		chatID: chatID,
		sender: sender,
		text:   message,
	}
}

func (u *UserHandler) SendMessage(ctx context.Context, req *rpc.SendMsgRequest) (*emptypb.Empty, error) {
	message := getMessageFromSendMessageRequest(req.GetChatId(), req.GetFrom(), req.GetText())

	builderInsert := sq.Insert("messages").
		PlaceholderFormat(sq.Dollar).
		Columns("id", "sender", "message_text").
		Values(message.chatID, message.sender, message.text)

	query, args, err := builderInsert.ToSql()
	if err != nil {
		log.Printf("message: %v, %v", message, errors.Wrap(err, "ToSql"))
		return nil, status.Error(codes.Internal, "preparing query")
	}

	ctxDB, cancel := context.WithTimeout(ctx, dbTimeOutDefault)
	defer cancel()

	tag, err := u.dbConn.Exec(ctxDB, query, args...)
	if err != nil {
		log.Printf("message: %v, %v", message, errors.Wrap(err, "QueryRow"))
		return nil, status.Error(codes.Internal, "writing in db")
	}

	if tag.RowsAffected() == 0 {
		return nil, status.Error(codes.Aborted, "record not set")
	}

	return &emptypb.Empty{}, nil
}

func (u *UserHandler) Delete(ctx context.Context, req *rpc.DeleteRequest) (*emptypb.Empty, error) {
	chatID := req.GetChatId()

	ctxDB, cancel := context.WithTimeout(ctx, dbTimeOutDefault)
	defer cancel()

	builderDelete := sq.Delete("chats").
		PlaceholderFormat(sq.Dollar).
		Where(sq.Eq{"id": chatID})

	query, args, err := builderDelete.ToSql()
	if err != nil {
		log.Printf("chatID: %v, %v", chatID, errors.Wrap(err, "ToSql"))
		return nil, status.Error(codes.Internal, "preparing query")
	}

	tag, err := u.dbConn.Exec(ctxDB, query, args...)
	if err != nil {
		log.Printf("chatID: %v, %v", chatID, errors.Wrap(err, "Exec"))
		return nil, status.Error(codes.Internal, "executing query")
	}

	if tag.RowsAffected() == 0 {
		return nil, status.Error(codes.NotFound, "record not found")
	}

	return &emptypb.Empty{}, nil
}
