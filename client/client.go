package client

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"os"
	"path/filepath"

	pb "github.com/Code-Hex/upload/internal/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/h2non/filetype.v1"
)

type Client struct {
	tlsPath string
}

func New() *Client {
	return &Client{
		tlsPath: "tls",
	}
}

func (c *Client) Run() int {
	if err := c.run(); err != nil {
		log.Println(err)
		return 1
	}
	return 0
}

func (c *Client) run() error {
	options := []grpc.DialOption{}
	crt := filepath.Join(c.tlsPath, "server.crt")
	key := filepath.Join(c.tlsPath, "server.key")
	cer, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		return errors.Wrap(err, "Failed to load credential key pair")
	}
	config := &tls.Config{
		Certificates:       []tls.Certificate{cer},
		InsecureSkipVerify: true,
	}
	options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(config)))

	conn, err := grpc.Dial(":12345", options...)
	if err != nil {
		return errors.Wrap(err, "Failed to connect to uploader")
	}
	defer conn.Close()

	return upload(pb.NewFileUploaderClient(conn), os.Args[1:]...)
}

func upload(cli pb.FileUploaderClient, files ...string) error {
	ctx := context.Background()
	for _, file := range files {
		stream, err := cli.Upload(ctx)
		if err != nil {
			return err
		}
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return err
		}
		if err := sendHeader(stream, f, info); err != nil {
			return err
		}
		if err := sendFile(stream, f, info.Size()); err != nil {
			return err
		}
		res, err := stream.CloseAndRecv()
		if err != nil {
			return err
		}
		if res.GetOk() == pb.StatusCodeType_Failed {
			return errors.New(res.GetErrMsg())
		}
		log.Printf("Send complete: %s\n", info.Name())
	}
	return nil
}

func sendFile(stream pb.FileUploader_UploadClient, f io.Reader, size int64) error {
	buf := make([]byte, size)
	for {
		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "Got unexpected error")
		}
		err = stream.Send(&pb.FileRequestType{
			File: &pb.FileRequestType_Chunk{
				Chunk: &pb.ChunkType{
					Data: buf[:n],
				},
			},
		})
		if err != nil {
			return errors.Wrap(err, "Failed to send file data")
		}
	}
	return nil
}

func sendHeader(stream pb.FileUploader_UploadClient, f *os.File, info os.FileInfo) error {
	h := MakeHeader(info)
	h.Add("content-type", mime(f, info))
	return stream.Send(&pb.FileRequestType{
		File: h.Cast(),
	})
}

func mime(f *os.File, info os.FileInfo) string {
	head := make([]byte, 261)
	f.Read(head)
	f.Seek(0, 0)
	kind, err := filetype.Match(head)
	if err != nil {
		return "unknown"
	}
	return kind.MIME.Value
}

type FileHeader pb.FileRequestType_Header

func MakeHeader(info os.FileInfo) *FileHeader {
	h := FileHeader(
		pb.FileRequestType_Header{
			Header: &pb.FileHeaderType{
				Name: info.Name(),
				Size: info.Size(),
			},
		},
	)
	return &h
}

func (f *FileHeader) Cast() *pb.FileRequestType_Header {
	if f == nil {
		return nil
	}
	h := pb.FileRequestType_Header(*f)
	return &h
}

func (f *FileHeader) Add(key, value string) {
	h := f.Header.Header
	for i := 0; i < len(h); i++ {
		if h[i].Key == key {
			h[i].Values = append(h[i].Values, value)
			return
		}
	}
	h = append(h, &pb.FileHeaderType_MIMEHeaderType{
		Key:    key,
		Values: []string{value},
	})
}

func (f *FileHeader) Get(key string) string {
	if f == nil {
		return ""
	}
	for _, h := range f.Header.Header {
		if h.Key == key {
			return h.Values[0]
		}
	}
	return ""
}
