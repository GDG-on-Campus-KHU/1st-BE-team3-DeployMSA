# main.tf

terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# GKE Autopilot cluster
resource "google_container_cluster" "autopilot" {
  name     = var.cluster_name
  location = var.region

  # Enable Autopilot mode
  enable_autopilot = true

  # Network configuration
  network    = "default"
  subnetwork = "default"

  # IP allocation policy for VPC-native cluster
  ip_allocation_policy {
    cluster_ipv4_cidr_block  = ""
    services_ipv4_cidr_block = ""
  }
}

# Configure kubernetes provider
data "google_client_config" "default" {}

provider "kubernetes" {
  host                   = "https://${google_container_cluster.autopilot.endpoint}"
  token                  = data.google_client_config.default.access_token
  cluster_ca_certificate = base64decode(google_container_cluster.autopilot.master_auth[0].cluster_ca_certificate)
}

# ConfigMaps
resource "kubernetes_config_map" "grpc_client_config" {
  metadata {
    name = "grpc-client-config"
  }
  depends_on = [ google_container_cluster.autopilot ]

  data = merge(var.grpc_client_config_defaults, var.grpc_client_config)
}

resource "kubernetes_config_map" "grpc_internal_config" {
  metadata {
    name = "grpc-internal-config"
  }
  depends_on = [ google_container_cluster.autopilot ]

  data = merge(var.grpc_internal_config_defaults, var.grpc_internal_config)
}

resource "kubernetes_config_map" "grpc_server_config" {
  metadata {
    name = "grpc-server-config"
  }
  depends_on = [ google_container_cluster.autopilot ]

  data = merge(var.grpc_server_config_defaults, var.grpc_server_config)
}

# Services
resource "kubernetes_service" "grpc_client_service" {
  metadata {
    name = "grpc-client-service" 
  }
  depends_on = [ kubernetes_config_map.grpc_client_config ]

  spec {
    selector = {
      app = "grpc-client"
    }

    port {
      protocol    = "TCP"
      port        = 5051    
      target_port = 50051
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_service" "grpc_server_service" {
  metadata {
    name = kubernetes_config_map.grpc_client_config.data.SERVER_HOST # grpc-client가 바라보는 grpc-server의 서비스 이름
  }
  depends_on = [ kubernetes_config_map.grpc_server_config ]

  spec {
    selector = {
      app = "grpc-server"
    }

    port {
      protocol    = "TCP"
      port        = kubernetes_config_map.grpc_client_config.data.SERVER_PORT
      target_port = kubernetes_config_map.grpc_server_config.data.SERVER_PORT
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_service" "grpc_internal_service" {
  metadata {
    name = kubernetes_config_map.grpc_server_config.data.INTERNAL_HOST # grpc-server가 바라보는 grpc-internal의 서비스 이름
  }
  
  depends_on = [ kubernetes_config_map.grpc_internal_config ]

  spec {
    selector = {
      app = "grpc-internal"
    }

    port {
      protocol    = "TCP"
      port        = kubernetes_config_map.grpc_server_config.data.INTERNAL_PORT
      target_port = kubernetes_config_map.grpc_internal_config.data.INTERNAL_PORT
    }

    type = "ClusterIP"
  }
}

# Deployments
resource "kubernetes_deployment" "grpc_client_deployment" {
  metadata {
    name = "grpc-client"
  }

  depends_on = [ kubernetes_config_map.grpc_client_config]

  spec {
    replicas = 3

    selector {
      match_labels = {
        app = "grpc-client"
      }
    }

    template {
      metadata {
        labels = {
          app = "grpc-client"
        }
      }

      spec {
        container {
          name  = "grpc-client"
          image = var.grpc_client_image

          env_from {
            config_map_ref {
              name = kubernetes_config_map.grpc_client_config.metadata[0].name
            }
          }

          port {
            container_port = 50051
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "grpc_server_deployment" {
  metadata {
    name = "grpc-server"
  }

  depends_on = [ kubernetes_config_map.grpc_server_config]

  spec {
    replicas = 3

    selector {
      match_labels = {
        app = "grpc-server"
      }
    }

    template {
      metadata {
        labels = {
          app = "grpc-server"
        }
      }

      spec {
        container {
          name  = "grpc-server"
          image = var.grpc_server_image

          env_from {
            config_map_ref {
              name = kubernetes_config_map.grpc_server_config.metadata[0].name
            }
          }

          port {
            container_port = kubernetes_config_map.grpc_server_config.data.SERVER_PORT
          }
        }
      }
    }
  }
}

resource "kubernetes_deployment" "grpc_internal_deployment" {
  metadata {
    name = "grpc-internal"
  }

  depends_on = [ kubernetes_config_map.grpc_internal_config]

  spec {
    replicas = 3

    selector {
      match_labels = {
        app = "grpc-internal"
      }
    }

    template {
      metadata {
        labels = {
          app = "grpc-internal"
        }
      }

      spec {
        container {
          name  = "grpc-internal"
          image = var.grpc_internal_image

          env_from {
            config_map_ref {
              name = kubernetes_config_map.grpc_internal_config.metadata[0].name
            }
          }
          resources {
            limits = {
              ephemeral-storage = "10Gi"  # GKE autopilot에서 최대 임시 저장소 크기. 더 큰 크기를 원한다면 pv, pvc 사용
              cpu    = "0.5"
              memory = "512Mi"              
            }
            requests = {
              ephemeral-storage = "10Gi"
              cpu    = "0.5"
              memory = "512Mi"
            }
          }

          port {
            container_port = kubernetes_config_map.grpc_internal_config.data.INTERNAL_PORT
          }
        }
      }
    }
  }
}