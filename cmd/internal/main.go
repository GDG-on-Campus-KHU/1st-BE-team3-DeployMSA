package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	pb "github.com/ket0825/grpc-streaming/api/proto"
	"github.com/lpernett/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type VideoQuality struct {
	Name      string
	Height    int
	Bitrate   string
	Directory string
}

var qualities = []VideoQuality{
	{Name: "1080p", Height: 1080, Bitrate: "5000k", Directory: "1080p"},
	{Name: "720p", Height: 720, Bitrate: "2500k", Directory: "720p"},
	{Name: "480p", Height: 480, Bitrate: "1000k", Directory: "480p"},
	{Name: "360p", Height: 360, Bitrate: "750k", Directory: "360p"},
}

type server struct {
	pb.UnimplementedVideoStreamingServiceServer
	mu                sync.Mutex
	activeProcessings map[string]*ProcessingInfo
}

type ProcessingInfo struct {
	file       *os.File
	totalBytes int64
	filename   string
}

func NewInternalServer() *server {
	return &server{
		activeProcessings: make(map[string]*ProcessingInfo),
	}
}

func (s *server) StreamVideo(stream pb.VideoStreamingService_StreamVideoServer) error {
	sessionID := fmt.Sprintf("process_%d", time.Now().UnixNano())
	log.Printf("Starting new processing session: %s", sessionID)

	// 임시 디렉토리 생성
	tempDir := "/Users/sangyeong_park/CE/Clubs/GDG_on_KHU/Go_Server/tempVideo"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	// 임시 파일 생성
	fileName := fmt.Sprintf("video_%s.mp4", sessionID)
	tempPath := filepath.Join(tempDir, fileName)
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}

	// 처리 정보 저장
	s.mu.Lock()
	s.activeProcessings[sessionID] = &ProcessingInfo{
		file:     file,
		filename: fileName,
	}
	s.mu.Unlock()

	defer func() {
		file.Close()
		s.mu.Lock()
		delete(s.activeProcessings, sessionID)
		s.mu.Unlock()
		// 임시 파일 삭제
		os.Remove(tempPath)
	}()

	// 청크 수신 및 파일 저장
	totalBytes := int64(0)
	chunks := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving chunk: %v", err)
		}

		n, err := file.Write(chunk.Data)
		if err != nil {
			return fmt.Errorf("failed to write chunk: %v", err)
		}

		totalBytes += int64(n)
		chunks++

		if chunks%1000 == 0 {
			log.Printf("Session %s: Received %d chunks, %d bytes",
				sessionID, chunks, totalBytes)
		}
	}

	log.Printf("Received complete video for %s: %d bytes in %d chunks",
		sessionID, totalBytes, chunks)

	// 파일을 닫고 다시 열어서 변환 시작
	file.Close()

	// 화질별 변환 시작
	baseDir := "/Users/sangyeong_park/CE/Clubs/GDG_on_KHU/Go_Server/savedVideo"
	successCount := 0
	var conversionErrors []string

	// 각 화질별로 변환
	for _, quality := range qualities {
		qualityDir := filepath.Join(baseDir, quality.Directory)
		if err := os.MkdirAll(qualityDir, 0755); err != nil {
			log.Printf("Failed to create directory for %s: %v", quality.Name, err)
			conversionErrors = append(conversionErrors,
				fmt.Sprintf("%s: directory creation failed", quality.Name))
			continue
		}

		outputPath := filepath.Join(qualityDir, fileName)
		if err := convertVideo(tempPath, outputPath, quality); err != nil {
			log.Printf("Failed to convert to %s: %v", quality.Name, err)
			conversionErrors = append(conversionErrors,
				fmt.Sprintf("%s: conversion failed", quality.Name))
			continue
		}

		successCount++
		log.Printf("Successfully converted to %s: %s", quality.Name, outputPath)
	}

	// 결과 메시지 생성
	var message string
	if successCount == len(qualities) {
		message = fmt.Sprintf("Successfully converted video to all %d qualities", successCount)
	} else if successCount > 0 {
		message = fmt.Sprintf("Partially converted video to %d/%d qualities. Errors: %v",
			successCount, len(qualities), conversionErrors)
	} else {
		message = fmt.Sprintf("Failed to convert video. Errors: %v", conversionErrors)
	}

	return stream.SendAndClose(&pb.StreamResponse{
		Success: successCount > 0,
		Message: message,
	})
}

func convertVideo(inputPath, outputPath string, quality VideoQuality) error {
	log.Printf("Converting to %s: %s", quality.Name, outputPath)

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%d", quality.Height),
		"-b:v", quality.Bitrate,
		"-c:v", "libx264",
		"-preset", "medium",
		"-c:a", "aac",
		"-b:a", "128k",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("conversion failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func main() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	// FFmpeg 설치 확인
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Fatal("FFmpeg is not installed. Please install FFmpeg first.")
	}

	INTERNAL_PORT := os.Getenv("INTERNAL_PORT")
	INTERNAL_HOST := os.Getenv("INTERNAL_HOST")
	internalAddr := fmt.Sprintf("%s:%s", INTERNAL_HOST, INTERNAL_PORT)

	lis, err := net.Listen("tcp", internalAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 15 * time.Second,
			MaxConnectionAge:  30 * time.Second,
			Time:              5 * time.Second,
			Timeout:           1 * time.Second,
		}),
		grpc.MaxRecvMsgSize(1024 * 1024 * 50), // 50MB
		grpc.MaxSendMsgSize(1024 * 1024 * 50), // 50MB
	}

	s := grpc.NewServer(opts...)
	pb.RegisterVideoStreamingServiceServer(s, NewInternalServer())

	log.Printf("Internal server listening at %v", internalAddr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
