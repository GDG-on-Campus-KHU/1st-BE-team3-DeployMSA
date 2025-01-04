# K8s를 활용한 gRPC 기반 스트리밍 서버 시스템
- 본 프로젝트는 gRPC를 활용하여 영상 데이터를 streaming하여 영상을 여러 화질로 인코딩하는 동작을 구현
- 추가적으로, googcloud의 GKE와 terraform을 활용하여 k8s 환경에 프로젝트를 구축함
- source code는 golang으로 작성
- 관리 및 배포를 위해 Docker, GKE(Google Kubernetes Engine), Terraform 등 활용

## **1. Service Architecture**
### **1) Service Architecture**
<img width="1000" alt="image" src="https://github.com/user-attachments/assets/7f4aa592-8f55-421a-b04f-f5424df2dfc8">

### **2) Service & Deployment 고려한 k8s architecture**
<img width="1000" alt="image" src="https://github.com/user-attachments/assets/c35e8a8d-417e-4bb0-80c7-ff3f01046079">


## **2. Main Components**

### **1) Client**

- **역할:**
    - HTTP 요청으로 Video Sample을 수신.
    - protobuf에 정의한 video chunk단위로 Video Sample을 stream 방식으로 server에 전달.

### **2) Server**

- **역할:**
    - Client로부터 video chunk 단위의 데이터를 stream 형태로 수신.
    - Internal로 video chunk 단위의 데이터를 stream 형태로 송신.
    - 중간 전달자 역할

### **3) Internal**

- **역할:**
    - Server로부터 video chunk 단위의 데이터를 stream 형태로 수신.
    - 전달받은 video chunk를 영상 파일로 인코딩.
    - 이때, 영상은 여려 화질의 파일로 나누어 저장.
 
## **3. Local 환경 실행 방법**

```
kubectl apply -f .\deploy\internal\config-grpc-internal.yaml
kubectl apply -f .\deploy\internal\k8s-grpc-internal.yaml
kubectl apply -f .\deploy\server\config-grpc-server.yaml
kubectl apply -f .\deploy\server\k8s-grpc-server.yaml
kubectl apply -f .\deploy\client\config-grpc-client.yaml
kubectl apply -f .\deploy\client\k8s-grpc-client.yaml

kubectl delete pods --all
kubectl delete deployments --all
kubectl delete services --all
kubectl delete pvc --all
kubectl delete pv --all
kubectl delete configmaps --all
```

## **4. GCP 환경 실행 방법(Terraform)**

```
gcloud auth application-default login
terraform init      // 패키지 의존성 등에 따른 설치 및 환경 시작
terraform plan      // 실제 배포 시에 변수 설정 확인 가능
terraform apply     // 실제 배포 과정 + 배포 후, cloudshell에서 로그 확인
terraform destroy   // 배포한 인프라 삭제
```

## **5. 추가 계획**

- `helm`: GKE 모니터링 시스템 구축
- `Prometheus` & `Grafana`: 서비스 모니터링 (metric data 수집 및 시각화)
