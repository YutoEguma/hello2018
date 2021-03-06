package client

import (
	"fmt"
	"io"

	pb "github.com/takaishi/hello2018/grpc_password_auth/protocol"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type loginCreds struct {
	Username, Password string
}

func (c *loginCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"username": c.Username,
		"password": c.Password,
	}, nil
}

func (c *loginCreds) RequireTransportSecurity() bool {
	return false
}

func dial(c *cli.Context) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(
			&loginCreds{
				Username: c.GlobalString("username"),
				Password: c.GlobalString("password"),
			},
		),
	}
	return grpc.Dial("127.0.0.1:11111", opts...)
}

func Add(c *cli.Context, name string, age int) error {
	conn, err := dial(c)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer conn.Close()

	client := pb.NewCustomerServiceClient(conn)

	person := &pb.Person{
		Name: name,
		Age:  int32(age),
	}
	_, err = client.AddPerson(context.Background(), person)
	return err
}

func List(c *cli.Context) error {
	conn, err := dial(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := pb.NewCustomerServiceClient(conn)

	stream, err := client.ListPerson(context.Background(), new(pb.RequestType))
	if err != nil {
		return err
	}
	for {
		person, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		fmt.Println(person)
	}
	return nil
}
