package uploader

import (
	"io"
	"log"
	"net"
	"os"
	"path/filepath"

	pb "github.com/Code-Hex/upload/internal/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Container struct {
	rootPath, tlsPath string
}

func NewService() *Container {
	return &Container{
		rootPath: "static",
		tlsPath:  "tls",
	}
}

func (c *Container) Run() int {
	if err := c.serve(); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

func (c *Container) serve() error {
	lis, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatalf("cannot listen: %v", err)
	}
	defer lis.Close()

	options := []grpc.ServerOption{}
	creds, err := credentials.NewServerTLSFromFile(
		filepath.Join(c.tlsPath, "server.crt"),
		filepath.Join(c.tlsPath, "server.key"))
	if err != nil {
		return err
	}
	options = append(options, grpc.Creds(creds))

	log.Println("server started:", lis.Addr().String())
	server := grpc.NewServer(options...)
	pb.RegisterFileUploaderServer(server, c)
	return server.Serve(lis)
}

func (c *Container) Upload(stream pb.FileUploader_UploadServer) error {
	req, err := stream.Recv()
	f, err := c.CreateFile(req)
	if err != nil {
		return c.Error(stream, err)
	}
	defer f.Close()
	if err := c.Recv(stream, f); err != nil {
		return c.Error(stream, err)
	}
	log.Printf("Uploaded: %s\n", req.GetHeader().GetName())
	return c.OK(stream)
}

func (c *Container) Recv(stream pb.FileUploader_UploadServer, f io.Writer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "Got unexpected error")
		}
		chunk := req.GetChunk()
		if _, err := f.Write(chunk.GetData()); err != nil {
			return errors.Wrap(err, "Failed to write received chunk data")
		}
	}
	return nil
}

// OK returns when success something processes
func (c *Container) OK(stream pb.FileUploader_UploadServer) error {
	return stream.SendAndClose(&pb.ResultResponseType{
		Ok: pb.StatusCodeType_OK,
	})
}

func (c *Container) Error(stream pb.FileUploader_UploadServer, err error) error {
	return stream.SendAndClose(&pb.ResultResponseType{
		Ok:     pb.StatusCodeType_Failed,
		ErrMsg: err.Error(),
	})
}

func (c *Container) CreateFile(req *pb.FileRequestType) (*os.File, error) {
	header := req.GetHeader()
	dest := filepath.Join(c.rootPath, header.GetName())
	f, err := os.Create(dest)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create file on server")
	}
	log.Println("File size:", header.GetSize())
	for _, dict := range header.GetHeader() {
		log.Println("Key:", dict.GetKey())
		for _, val := range dict.GetValues() {
			log.Println("    :", val)
		}
	}
	return f, nil
}
