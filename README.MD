# CS6650 HW5: Product API with Terraform, Load Testing, and Observability

## Project Structure

```
src/
  product-service.go    # Product API server (Gin framework)
  Dockerfile            # Multi-stage Docker build
  go.mod / go.sum       # Go dependencies
terraform/
  main.tf               # Wires together all modules
  variables.tf          # Configurable values (port, region, etc)
  provider.tf           # AWS and Docker providers
  output.tf             # Cluster and service names
  modules/
    ecr/                # ECR repository
    ecs/                # ECS cluster, task definition, service
    network/            # VPC, subnets, security group
    logging/            # CloudWatch log group
```

## Deployment Instructions

### Prerequisites
- AWS CLI installed
- Terraform installed
- Docker Desktop running

### Steps

1. Get credentials from Learner Lab and set them:
```bash
export AWS_ACCESS_KEY_ID=<your key>
export AWS_SECRET_ACCESS_KEY=<your secret>
export AWS_SESSION_TOKEN=<your token>
export AWS_DEFAULT_REGION=us-west-2
export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock
```

2. Deploy:
```bash
cd terraform
terraform init
terraform apply -auto-approve
```

3. Get the public IP:
```bash
aws ec2 describe-network-interfaces \
--network-interface-ids $(
    aws ecs describe-tasks \
    --cluster $(terraform output -raw ecs_cluster_name) \
    --tasks $(
        aws ecs list-tasks \
        --cluster $(terraform output -raw ecs_cluster_name) \
        --service-name $(terraform output -raw ecs_service_name) \
        --query 'taskArns[0]' --output text
    ) \
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
    --output text
) \
--query 'NetworkInterfaces[0].Association.PublicIp' \
--output text
```

4. Test:
```bash
curl http://<PUBLIC_IP>:8080/products/1
```

5. Clean up:
```bash
terraform destroy -auto-approve
```

## API Endpoints and Response Codes

### GET /products/{productId}

**200: Product found**
```bash
curl http://localhost:8080/products/1
```
```json
{"product_id":1,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":10,"weight":1250,"some_other_id":100}
```

**404: Product not found**
```bash
curl http://localhost:8080/products/999
```
```json
{"error":"NOT_FOUND","message":"Product not found","details":"No product with ID 999"}
```

### POST /products/{productId}/details

**204: Product details updated**
```bash
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"sku":"NEW-SKU","manufacturer":"NewCorp","category_id":5,"weight":100,"some_other_id":50}'
```
No response body (204 No Content).

**400: Invalid input (empty required field)**
```bash
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d '{"sku":"","manufacturer":"NewCorp","category_id":5,"weight":100,"some_other_id":50}'
```
```json
{"error":"INVALID_INPUT","message":"Missing required field","details":"sku is required and cannot be empty"}
```

**400: Invalid JSON**
```bash
curl -X POST http://localhost:8080/products/1/details \
  -H "Content-Type: application/json" \
  -d 'not json'
```
```json
{"error":"INVALID_INPUT","message":"Invalid request body","details":"invalid character 'o' in literal null (expecting 'u')"}
```

**404: Product not found**
```bash
curl -X POST http://localhost:8080/products/999/details \
  -H "Content-Type: application/json" \
  -d '{"sku":"X","manufacturer":"Y","category_id":1,"weight":1,"some_other_id":1}'
```
```json
{"error":"NOT_FOUND","message":"Product not found","details":"No product with ID 999"}
```

## Part IV: Load Testing with Locust

### Test 1: Normal Load (50 users, HttpUser)
- 128 RPS, 0 failures, ~90ms average response time
- 3:1 GET to POST ratio

### Test 2: Stress Test (500 users, HttpUser)
- 1258 RPS, 0 failures, ~96ms average response time
- Server handled the load without breaking
- p99 latency increased from 180ms to 200ms, max hit 1110ms

### Test 3: FastHttpUser (50 users)
- 127 RPS, 0 failures, ~90ms average response time
- Nearly identical to HttpUser
- The server is the bottleneck, not the load generator. FastHttpUser reduces Python overhead but since the server caps at ~128 RPS with 50 users, a faster client makes no difference.

Screenshots included in the repo.

## Part V: Observability

### Prometheus Metrics
The server exposes a /metrics endpoint with three RED signals:
- `http_requests_total`: counter for request rate
- `http_errors_total`: counter for non-2xx responses
- `http_request_duration_seconds`: histogram for latency

### Grafana Dashboard
Three panels: Request Rate, Error Rate, Response Duration (p95). The dashboard shows clear spikes during load testing compared to idle state.

### Structured JSON Logs
Each request emits a JSON log line:
```json
{"timestamp":"2026-10-18T23:51:14Z","method":"GET","endpoint":"/products/:productId","status_code":200,"duration_ms":0.004}
```

### When to use dashboard vs logs
Use the Grafana dashboard for live monitoring: watching traffic patterns, spotting spikes in errors or latency as they happen, and comparing current behavior to baseline. Use the logs for after the fact debugging: when something went wrong and you need to trace specific requests, find out which endpoints failed, or check exact timestamps and durations for individual calls.
