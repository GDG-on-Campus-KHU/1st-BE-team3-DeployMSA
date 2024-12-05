# outputs.tf

output "cluster_endpoint" {
  description = "Cluster endpoint"
  value       = google_container_cluster.autopilot.endpoint
}

output "cluster_name" {
  description = "Cluster name"
  value       = google_container_cluster.autopilot.name
}

output "region" {
  description = "GCP region"
  value       = var.region
}