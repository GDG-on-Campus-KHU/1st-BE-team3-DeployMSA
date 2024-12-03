resource "google_container_cluster" "primary" {
    name    = "my-gke-cluster:
    location = "us-central1-c"
    initial_node_count = 1 # 각 node count
    enable-autopilot = true // 오토파일럿 활성화. 노드는 사용자가 관리할 필요가 없음. 비용 청구 x. 노출도 하지 않음.
    끄면 노드를 우리가 관리해야함.
    // pod만 따로 따로 과금됨.
    // Tenant isolation를 하려면 Namespace를 생성해야함.
}

