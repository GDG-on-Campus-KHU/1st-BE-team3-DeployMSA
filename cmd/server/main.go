package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/ket0825/grpc-streaming/api/proto"
	"github.com/lpernett/godotenv"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type VideoStreamingServer struct {
	pb.UnimplementedVideoStreamingServiceServer
	mu            sync.Mutex
	activeStreams map[string]*StreamInfo
}

type StreamInfo struct {
	file     *os.File
	chunks   int
	bytesCnt int64
}

func NewVideoStreamingServer() *VideoStreamingServer {
	return &VideoStreamingServer{
		activeStreams: make(map[string]*StreamInfo),
	}
}

func (s *VideoStreamingServer) StreamVideo(stream pb.VideoStreamingService_StreamVideoServer) error {
	// 스트림 ID 생성 (실제로는 더 강력한 ID 생성 방식을 사용해야 함)
	streamID := fmt.Sprintf("stream_%d", time.Now().UnixNano())

	// 저장할 디렉토리 생성
	outputDir := os.Getenv("OUTPUT_DIR")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 출력 파일 생성
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", streamID))
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// 스트림 정보 초기화
	s.mu.Lock()
	s.activeStreams[streamID] = &StreamInfo{
		file:     file,
		chunks:   0,
		bytesCnt: 0,
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeStreams, streamID)
		s.mu.Unlock()
	}()

	log.Printf("Started receiving stream: %s", streamID)

	var lastSequence int32 = -1
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// 스트리밍 완료
			log.Printf("Stream completed: %s, received %d chunks, %d bytes",
				streamID, s.activeStreams[streamID].chunks, s.activeStreams[streamID].bytesCnt)

			return stream.SendAndClose(&pb.StreamResponse{
				Success: true,
				Message: fmt.Sprintf("Successfully received %d chunks", s.activeStreams[streamID].chunks),
			})
		}
		if err != nil {
			return fmt.Errorf("error receiving chunk: %w", err)
		}

		// 시퀀스 번호 확인
		if chunk.Sequence != lastSequence+1 {
			log.Printf("Warning: Received out-of-order chunk. Expected %d, got %d",
				lastSequence+1, chunk.Sequence)
		}
		lastSequence = chunk.Sequence

		// 데이터 쓰기
		n, err := file.Write(chunk.Data)
		if err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}

		// 스트림 정보 업데이트
		s.mu.Lock()
		if info, exists := s.activeStreams[streamID]; exists {
			info.chunks++
			info.bytesCnt += int64(n)
		}
		s.mu.Unlock()

		// 주기적으로 진행상황 로깅
		if chunk.Sequence%100 == 0 {
			log.Printf("Stream %s: Received chunk %d, total bytes: %d",
				streamID, chunk.Sequence, s.activeStreams[streamID].bytesCnt)
		}
	}
}

// 서버 상태 모니터링을 위한 메서드
func (s *VideoStreamingServer) GetActiveStreams() map[string]StreamInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]StreamInfo)
	for id, info := range s.activeStreams {
		result[id] = *info
	}
	return result
}

// TLS 없이 HTTP/2를 사용하려면 (h2c)
func serveH2C(port string, server *grpc.Server) error {
	h2Handler := h2c.NewHandler(server, &http2.Server{
		MaxConcurrentStreams: 100,
		MaxReadFrameSize:     1024 * 1024,
		IdleTimeout:          30 * time.Second,
	})

	return http.ListenAndServe(port, h2Handler)
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")
	port = fmt.Sprintf(":%s", port)

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// HTTP/2 설정
	// var opts []grpc.ServerOption

	// TLS 설정 (선택적)
	// TLS를 사용하는 경우
	/*
		 cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		 if err != nil {
			 log.Fatalf("Failed to load cert: %v", err)
		 }

		 creds := credentials.NewTLS(&tls.Config{
			 Certificates: []tls.Certificate{cert},
			 NextProtos:   []string{"h2"}, // HTTP/2 프로토콜 명시
		 })
		 opts = append(opts, grpc.Creds(creds))
	*/

	// HTTP/2 관련 서버 옵션 설정
	// opts = append(opts,
	// 	// 최대 메시지 크기 설정
	// 	grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
	// 	grpc.MaxSendMsgSize(10*1024*1024), // 10MB

	// 	// HTTP/2 keepalive 설정
	// 	grpc.KeepaliveParams(keepalive.ServerParameters{
	// 		MaxConnectionIdle:     15 * time.Second, // 유휴 연결 최대 시간
	// 		MaxConnectionAge:      30 * time.Second, // 연결 최대 수명
	// 		MaxConnectionAgeGrace: 5 * time.Second,  // 강제 종료 전 유예 기간
	// 		Time:                  5 * time.Second,  // ping 간격
	// 		Timeout:               1 * time.Second,  // ping 타임아웃
	// 	}),

	// 	// 클라이언트 keepalive 설정
	// 	grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
	// 		MinTime:             5 * time.Second, // 최소 keepalive 간격
	// 		PermitWithoutStream: true,            // 스트림 없이도 keepalive 허용
	// 	}),

	// 	// HTTP/2 전송 설정
	// 	grpc.InitialConnWindowSize(int32(1024*1024)), // 초기 연결 윈도우 크기
	// 	grpc.InitialWindowSize(int32(1024*1024)),     // 초기 스트림 윈도우 크기
	// 	grpc.MaxConcurrentStreams(uint32(100)),       // 동시 스트림 최대 개수
	// )

	// HTTP/2 관련 서버 옵션 설정
	opts := []grpc.ServerOption{
		// 최대 메시지 크기 설정
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB
		grpc.MaxSendMsgSize(10 * 1024 * 1024), // 10MB

		// HTTP/2 keepalive 설정
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Second,
			Time:              5 * time.Second,
			Timeout:           1 * time.Second,
		}),
	}

	// gRPC 서버 생성
	server := grpc.NewServer(opts...)

	streamingServer := NewVideoStreamingServer()
	pb.RegisterVideoStreamingServiceServer(server, streamingServer)

	// 상태 모니터링을 위한 고루틴
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			streams := streamingServer.GetActiveStreams()
			log.Printf("Active streams: %d", len(streams))
			for id, info := range streams {
				log.Printf("Stream %s: %d chunks, %d bytes",
					id, info.chunks, info.bytesCnt)
			}
		}
	}()

	log.Printf("Server started on port %s", port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
