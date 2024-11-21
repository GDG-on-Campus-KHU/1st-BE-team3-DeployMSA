package streamer

import (
	"context"
	"fmt"
	"io"

	pb "github.com/ket0825/grpc-streaming/api/proto"
	"github.com/ket0825/grpc-streaming/internal/client/fetcher"
	"google.golang.org/grpc"
)

type GRPCStreamer struct {
	client     pb.VideoStreamingServiceClient
	bufferSize int
}

func NewGRPCStreamer(conn *grpc.ClientConn) *GRPCStreamer {
	return &GRPCStreamer{
		client:     pb.NewVideoStreamingServiceClient(conn),
		bufferSize: 32 * 1024, // 32KB buffer
	}
}

// VideoResponse는 비디오 스트리밍 응답을 나타냅니다.

func (s *GRPCStreamer) StreamToServer(ctx context.Context, videoResp *fetcher.VideoResponse) error {
	stream, err := s.client.StreamVideo(ctx)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.CloseSend()

	sequence := 0
	buffer := make([]byte, s.bufferSize)

	headers := make(map[string]string)
	for k, v := range videoResp.Headers {
		headers[k] = v[0]
	}

	// 비디오 스트리밍을 청크 단위로 전송
	// flow control 로직 필요
	for {
		n, err := videoResp.Body.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading video chunk: %w", err)
		}

		chunk := &pb.VideoChunk{
			Data:        buffer[:n],
			ContentType: videoResp.ContentType,
			Headers:     headers,
			Sequence:    int32(sequence),
		}
		if err := stream.Send(chunk); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}

		sequence++
	}

	response, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("error receiving response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("streaming failed: %s", response.Message)
	}

	return nil
}
