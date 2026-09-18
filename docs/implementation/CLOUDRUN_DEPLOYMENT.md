# Deploying Go gRPC to Google Cloud Run (No Local Docker)

This guide walks you through deploying your Go gRPC server application directly from your source code using **Google Cloud Buildpacks** and **Google Cloud Run**. It configures end-to-end HTTP/2 for gRPC, uses zero local Docker dependencies, and manages configuration using standard variables (Option 1) and secure secrets (Option 3).

---

## 📋 Prerequisites

1. **Google Cloud SDK**: Install the `gcloud` CLI on your machine.
2. **Authenticated CLI**: Run `gcloud auth login` and `gcloud config set project YOUR_PROJECT_ID`.
3. **Enabled APIs**: Ensure the necessary services are active in your GCP project:
   ```bash
   gcloud services enable ://googleapis.com \
                          ://googleapis.com \
                          ://googleapis.com \
                          ://googleapis.com
   ```

---

## 🛠️ Step 1: Prepare the Go Application

### 1. Match the Network Port
Cloud Run injects a dynamic `$PORT` environment variable. Your `main.go` **must** listen on `0.0.0.0` and look for this variable.

```go
package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Fallback for local development
	}

	// Must bind to 0.0.0.0 for Cloud Run routing
	address := fmt.Sprintf("0.0.0.0:%s", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	// ... Register your gRPC services here ...

	log.Printf("gRPC server listening on %s", address)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
```

### 2. Create a `.gcloudignore` File
Create a `.gcloudignore` file in your root folder to avoid uploading local binaries or node modules to the cloud builder:

```text
.gcloudignore
.git
.gitignore
bin/
*.exe
*.out
```

---

## 🔐 Step 2: Set Up Secure Secrets (Option 3)

For sensitive data (like database credentials or API keys), use **Google Secret Manager**.

1. **Create the secret** in your GCP project:
   ```bash
   echo -n "your-super-secret-database-password" | gcloud secrets create DB_PASSWORD --data-file=-
   ```

2. **Authorize Cloud Run** to access the secret. Cloud Run uses your project's default compute service account to pull secrets:
   ```bash
   # Retrieve your project number
   PROJECT_NUMBER=$(gcloud projects describe $(gcloud config get-value project) --format="value(projectNumber)")

   # Grant access permission
   gcloud secrets add-iam-policy-binding DB_PASSWORD \
     --member="serviceAccount:${PROJECT_NUMBER}-compute@://gserviceaccount.com" \
     --role="roles/secretmanager.secretAccessor"
   ```

---

## 🚀 Step 3: Deploy to Cloud Run

Run the single-line deployment command below. This leverages **Google Buildpacks** to compile your binary remotely in the cloud and sets your configuration parameters.

### Execution Command

```bash
gcloud run deploy my-grpc-service \
  --source . \
  --use-http2 \
  --port 8080 \
  --allow-unauthenticated \
  --set-env-vars APP_ENV=production,LOG_LEVEL=info \
  --update-secrets=DATABASE_URL=DB_PASSWORD:latest
```

### 🔍 Flag Breakdown:
* `--source .`: Triggers remote Cloud Buildpacks. Google reads your `go.mod`, compiles your binary, packages it, and uploads it. **No local Docker engine is required.**
* `--use-http2`: **Critical for gRPC.** Forces Cloud Run to preserve end-to-end HTTP/2 multi-plexing instead of downgrading to HTTP/1.1.
* `--set-env-vars`: **(Option 1)** Injects standard, non-sensitive environment configuration arrays.
* `--update-secrets`: **(Option 3)** Securely binds your Secret Manager values to environment variables (`DATABASE_URL`) inside the container at runtime.

*Note: If your `main.go` file is located inside a nested subdirectory (e.g., `cmd/server/main.go`), append this flag to the deploy command:*
```bash
  --set-env-vars GO_TARGET=./cmd/server
```

---

## 🧪 Step 4: Verify the Connection

Cloud Run will output a service URL once the deployment completes successfully (e.g., `https://run.app`). 

Because Google's edge load balancer terminates SSL, you connect to the host over public **port 443** using standard transport security credentials (TLS), while your container processes it natively.

### Test with `grpcurl`:
```bash
grpcurl my-grpc-service-xyz.run.app:443 list
```
