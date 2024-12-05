# K8s를 활용한 gRPC 기반 스트리밍 서버 시스템
본 프로젝트는 gRPC를 활용하여 bidrectional 형태로 영상 데이터를 streaming하여 영상을 여러 화질로 인코딩하는 동작을 구현했습니다. 
추가적으로, googcloud의 GKE와 terraform을 활용하여 k8s 환경에 프로젝트를 구축했습니다.

## Service Architecture
<img width="447" alt="image" src="https://github.com/user-attachments/assets/7f4aa592-8f55-421a-b04f-f5424df2dfc8">
