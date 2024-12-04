package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/ket0825/grpc-streaming/api/proto"
	"github.com/lpernett/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type VideoStreamingServer struct {
	pb.UnimplementedVideoStreamingServiceServer
	mu             sync.Mutex
	activeStreams  map[string]*StreamInfo
	internalClient pb.VideoStreamingServiceClient
}

type StreamInfo struct {
	chunks   int
	bytesCnt int64
}

func NewVideoStreamingServer(internalClient pb.VideoStreamingServiceClient) *VideoStreamingServer {
	return &VideoStreamingServer{
		activeStreams:  make(map[string]*StreamInfo),
		internalClient: internalClient,
	}
}

func (s *VideoStreamingServer) StreamVideo(stream pb.VideoStreamingService_StreamVideoServer) error {
	// Internal 서버와의 스트리밍 시작
	ctx := context.Background()

	internalStream, err := s.internalClient.StreamVideo(ctx)
	if err != nil {
		log.Printf("Internal server connection failed: %v", err)
		return err
	}

	streamID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
	s.mu.Lock()
	s.activeStreams[streamID] = &StreamInfo{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeStreams, streamID)
		s.mu.Unlock()
	}()

	log.Printf("Started new stream: %s", streamID)

	// 데이터 스트리밍
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving chunk: %v", err)
		}

		// Internal 서버로 청크 전송
		if err := internalStream.Send(chunk); err != nil {
			return fmt.Errorf("failed to send chunk to internal: %v", err)
		}

		s.mu.Lock()
		if info := s.activeStreams[streamID]; info != nil {
			info.chunks++
			info.bytesCnt += int64(len(chunk.Data))
			if info.chunks%1000 == 0 {
				log.Printf("Stream %s: Received %d chunks, %d bytes",
					streamID, info.chunks, info.bytesCnt)
			}
		}
		s.mu.Unlock()
	}

	// Internal 서버로부터 응답 받기
	response, err := internalStream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to get internal response: %v", err)
	}

	log.Printf("Stream %s completed: %s", streamID, response.Message)

	// 클라이언트에 응답
	return stream.SendAndClose(&pb.StreamResponse{
		Success: response.Success,
		Message: response.Message,
	})
}

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 환경변수에서 Internal 서버 주소 가져오기
	internalHost := os.Getenv("INTERNAL_HOST")
	internalPort := os.Getenv("INTERNAL_PORT")
	internalAddr := fmt.Sprintf("%s:%s", internalHost, internalPort)

	// Internal 서버 연결
	internalConn, err := grpc.Dial(
		internalAddr,
		grpc.WithInsecure(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             2 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to connect to internal server: %v", err)
	}
	defer internalConn.Close()

	internalClient := pb.NewVideoStreamingServiceClient(internalConn)

	// 서버 설정
	SERVER_PORT := os.Getenv("SERVER_PORT")
	SERVER_HOST := os.Getenv("SERVER_HOST")
	serverAddr := fmt.Sprintf("%s:%s", SERVER_HOST, SERVER_PORT)
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Second,
			Time:              5 * time.Second,
			Timeout:           1 * time.Second,
		}),
	}

	server := grpc.NewServer(opts...)
	pb.RegisterVideoStreamingServiceServer(server, NewVideoStreamingServer(internalClient))

	log.Printf("Server started on %s", serverAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
