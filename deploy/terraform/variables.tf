# variables.tf

variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
}

variable "region" {
  description = "GCP region for the cluster"
  type        = string
  default     = "us-central1"
}

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
  default     = "grpc-cluster-autopilot"
}

variable "grpc_client_image" {
  description = "Docker image for gRPC client"
  type        = string
}

variable "grpc_internal_image" {
  description = "Docker image for gRPC internal service"
  type        = string
}

variable "grpc_server_image" {
  description = "Docker image for gRPC server"
  type        = string
}

variable "grpc_client_config_defaults" {
  description = "Default gRPC Client ConfigMap data"
  type = map(string)
  default = {
    VIDEO_URL   = "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4"
    SERVER_HOST = "grpc-server-service"
    SERVER_PORT = "5052"
  }
}

variable "grpc_client_config" {
  description = "gRPC Client ConfigMap data"
  type = map(string)
  default = {}
}

variable "grpc_internal_config_defaults" {
  description = "Default gRPC Internal ConfigMap data"
  type = map(string)
  default = {
    INTERNAL_PORT = "50053"
    INTERNAL_HOST = "0.0.0.0"
    OUT_DIR      = "../../encoded_videos"
    TEMP_DIR     = "../../temp"
  }
}

variable "grpc_internal_config" {
  description = "gRPC Internal ConfigMap data"
  type = map(string)
  default = {}
}

variable "grpc_server_config_defaults" {
  description = "Default gRPC Server ConfigMap data"
  type = map(string)
  default = {
    SERVER_HOST   = "0.0.0.0"
    SERVER_PORT   = "50052"
    INTERNAL_PORT = "5053"
    INTERNAL_HOST = "grpc-internal-service"
  }
}

variable "grpc_server_config" {
  description = "gRPC Server ConfigMap data"
  type = map(string)
  default = {}
}