package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	pb "github.com/ket0825/grpc-streaming/api/proto"
	"github.com/lpernett/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 추가할 수 있는 개선사항:

// 버퍼 크기 설정 옵션
// 재시도 메커니즘
// 진행률 모니터링
// 메모리 사용량 최적화
// 에러 복구 전략

// import (
// 	"context"
// 	"log"
// 	"time"

// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/credentials/insecure"

// 	"github.com/ket0825/grpc-streaming/internal/client/fetcher"
// 	"github.com/ket0825/grpc-streaming/internal/client/streamer"
// )

// func main() {
// 	conn, err := grpc.NewClient(
// 		"localhost:50051", // gRPC 서버 주소
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		grpc.WithBlock(),
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to connect server: %v", err)
// 	}
// 	defer conn.Close()

// 	// 비디오 fetcher 생성
// 	videoFetcher := fetcher.NewHTTPVideoFetcher()

// 	// gRPC streamer 생성
// 	grpcStreamer := streamer.NewGRPCStreamer(conn)

// 	// context 생성 (5분 타임아웃)
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
// 	defer cancel()

// 	// 비디오 URL
// 	videoURL := "http://commondatastorage.googleapis.com/gtv-videos-bucket"

// 	// 비디오 fetch
// 	videoResp, err := videoFetcher.Fetch(videoURL)
// 	if err != nil {
// 		log.Fatalf("Failed to fetch video: %v", err)
// 	}
// 	defer videoResp.Body.Close()

// 	// 서버로 스트리밍
// 	if err := grpcStreamer.StreamToServer(ctx, videoResp); err != nil {
// 		log.Fatalf("Failed to stream to server: %v", err)
// 	}

// 	log.Println("Successfully streamed video to server")
// }

func continuousStreamVideo(ctx context.Context, conn *grpc.ClientConn, videoURL string) error {
	// gRPC 클라이언트 설정
	grpcClient := pb.NewVideoStreamingServiceClient(conn)

	// HTTP 클라이언트 설정 (타임아웃 설정)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 시퀀스 번호 관리
	sequence := 0

	// gRPC 스트림 시작
	stream, err := grpcClient.StreamVideo(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.CloseSend()

	// 무한 루프로 계속 스트리밍
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// HTTP 요청 생성
			req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			// Range 헤더를 사용하여 부분 요청 가능
			// req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))

			// HTTP 응답 받기
			resp, err := httpClient.Do(req)
			if err != nil {
				log.Printf("Error getting response: %v, retrying...", err)
				time.Sleep(time.Second) // 에러 시 잠시 대기
				continue
			}

			// 버퍼 설정
			buffer := make([]byte, 32*1024) // 32KB 버퍼

			// 헤더 정보 변환
			headers := make(map[string]string)
			for k, v := range resp.Header {
				if len(v) > 0 {
					headers[k] = v[0]
				}
			}

			// 청크 단위로 읽어서 바로 전송
			for {
				n, err := resp.Body.Read(buffer)
				if err == io.EOF {
					break // 현재 응답 데이터를 모두 읽음
				}
				if err != nil {
					resp.Body.Close()
					log.Printf("Error reading chunk: %v, reconnecting...", err)
					break // 다음 요청으로 넘어감
				}

				// 청크 생성 및 전송
				chunk := &pb.VideoChunk{
					Data:        buffer[:n],
					ContentType: resp.Header.Get("Content-Type"),
					Headers:     headers,
					Sequence:    int32(sequence),
				}

				if err := stream.Send(chunk); err != nil {
					resp.Body.Close()
					return fmt.Errorf("failed to send chunk: %w", err)
				}

				sequence++
				log.Printf("Sent chunk %d, size: %d bytes", sequence, n)
			}

			resp.Body.Close()
		}
	}
}

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	videoURL := os.Getenv("VIDEO_URL")
	serverHost := os.Getenv("SERVER_HOST")
	serverPort := os.Getenv("SERVER_PORT")

	grpcAddr := fmt.Sprintf("%s:%s", serverHost, serverPort)

	maxMsgSize := 10 * 1024 * 1024 // 10MB (서버와 동일하게)

	conn, err := grpc.Dial(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(maxMsgSize),
			grpc.MaxCallRecvMsgSize(maxMsgSize),
		),
	)
	if err != nil {
		log.Fatalf("Failed to connect server: %v", err)
	}
	defer conn.Close()

	// 컨텍스트 설정 (필요한 경우 타임아웃 설정)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 스트리밍 시작
	if err := continuousStreamVideo(ctx, conn, videoURL); err != nil {
		if err == context.Canceled {
			log.Println("Streaming was canceled")
		} else {
			log.Fatalf("Streaming failed: %v", err)
		}
	}

}
