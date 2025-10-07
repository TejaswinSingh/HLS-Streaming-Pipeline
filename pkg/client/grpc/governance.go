package grpc

import (
	"context"
	"time"

	pb "github.com/Resolyte-official/gen_proto/pkg/governance"
	"github.com/Resolyte-official/service-gps/pkg/contextkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

type GovernanceClient interface {
	ServiceStatus(ctx context.Context) (*pb.ServiceStatusResponse, error)
	Authenticate(ctx context.Context, token string) (*pb.AuthenticateResponse, error)
	Authorize(ctx context.Context, token, obj, act string) (*pb.AuthorizeResponse, error)
	Close() error
}

type governanceClient struct {
	conn       *grpc.ClientConn
	grpcClient pb.GovernanceClient
	timeout    time.Duration
}

const requestTimeout = 2 * time.Second

func NewGovernanceClient(address string, creds credentials.TransportCredentials) (GovernanceClient, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &governanceClient{
		conn:       conn,
		grpcClient: pb.NewGovernanceClient(conn),
		timeout:    requestTimeout,
	}, nil
}

func (c *governanceClient) Close() error {
	return c.conn.Close()
}

func (c *governanceClient) contextWithMetadata(ctx context.Context) (context.Context, context.CancelFunc) {
	reqID, _ := ctx.Value(contextkeys.RequestIDKey).(string)
	md := metadata.New(map[string]string{
		"X-REQUEST-ID": reqID,
	})
	newCtx := metadata.NewOutgoingContext(ctx, md)
	return context.WithTimeout(newCtx, c.timeout)
}

func (c *governanceClient) ServiceStatus(ctx context.Context) (*pb.ServiceStatusResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.ServiceStatus(ctx, &pb.ServiceStatusRequest{})
}

func (c *governanceClient) Authenticate(ctx context.Context, token string) (*pb.AuthenticateResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.Authenticate(ctx, &pb.AuthenticateRequest{
		Token: token,
	})
}

func (c *governanceClient) Authorize(ctx context.Context, token, obj, act string) (*pb.AuthorizeResponse, error) {
	ctx, cancel := c.contextWithMetadata(ctx)
	defer cancel()
	return c.grpcClient.Authorize(ctx, &pb.AuthorizeRequest{
		Token: token,
		Obj:   obj,
		Act:   act,
	})
}
