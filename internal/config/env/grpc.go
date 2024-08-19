package env

import (
	"errors"
	"net"
	"os"

	"github.com/Zzarin/chat-server/internal/config"
)

const (
	grpcHostEnvName = "GRPC_HOST"
	grpcPortEnvName = "GRPC_PORT"
)

var _ config.GRPCConfig = (*grpcConfig)(nil)

type grpcConfig struct {
	host string
	port string
}

func NewGRPCConfig() (config.GRPCConfig, error) {
	host := os.Getenv(grpcHostEnvName)
	if len(host) == 0 {
		return nil, errors.New("grpc host is empty")
	}

	port := os.Getenv(grpcPortEnvName)
	if len(host) == 0 {
		return nil, errors.New("grpc port is empty")
	}

	return &grpcConfig{
		host: host,
		port: port,
	}, nil
}

func (g *grpcConfig) GetAddress() string {
	return net.JoinHostPort(g.host, g.port)
}
