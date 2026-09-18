# Google Cloud Platform (GCP) Console Setup Guide

This document covers all the necessary steps to configure your Google Cloud Console for hosting a Docker-free Go gRPC server on **Cloud Run** using **Cloud Buildpacks** and **Secret Manager**.

---

## 🛠️ Step 1: Enable Cloud APIs

Before deploying, your GCP project must have access to the underlying infrastructure services. You can enable these via the web console or instantly through your terminal.

### Option A: Via the Terminal (Fastest)
Run the following command in your terminal using the `gcloud` CLI:
```bash
gcloud services enable ://googleapis.com \
                       ://googleapis.com \
                       ://googleapis.com \
                       ://googleapis.com
```

### Option B: Via the GCP Web Console
1. Navigate to the **Google Cloud Console**.
2. Using the search bar at the top, find and select **APIs & Services > Library**.
3. Search for and click **Enable** on the following four APIs:
   * 🖥️ **Cloud Run API** (`://googleapis.com`)
   * 🏗️ **Cloud Build API** (`://googleapis.com`)
   * 📦 **Artifact Registry API** (`://googleapis.com`)
   * 🔐 **Secret Manager API** (`://googleapis.com`)

---

## 🔐 Step 2: Configure Secret Manager IAM Permissions

By default, Cloud Run instances run under your project's **Default Compute Service Account**. To allow your Go server to read secure environment strings at runtime, you must grant this service account permission to access **Secret Manager**.

1. In the GCP Console, go to **IAM & Admin > IAM** using the left-hand navigation menu.
2. Look through the list of principals to locate your project's default service account. It will follow this exact pattern:
   ```text
   YOUR_PROJECT_NUMBER-compute@://gserviceaccount.com
   ```
3. Click the ✏️ **Edit Principal** (pencil icon) on the far right of that row.
4. Click **Add Another Role**.
5. Search for and select: **`Secret Manager Secret Accessor`**.
6. Click **Save**.

---

## 🏗️ Step 3: Verify Artifact Registry Setup

When you invoke a source-based build (`gcloud run deploy --source .`), Google Cloud Build compiles your code and automatically outputs a Docker container. 

1. Cloud Run automatically checks for a repository named `cloud-run-source-deploy` in your target deployment region (e.g., `us-central1`).
2. **No action is required from you:** Google creates this repository dynamically during your first deployment run. 
3. *Note: Ensure your GCP project has a linked billing account, as Cloud Build and Artifact Registry require a valid billing method to execute storage operations.*

---

## 🏁 Summary Checklist Before Your First Test Deploy

| Component | Status | Purpose |
| :--- | :--- | :--- |
| **API Activation** | Enabled | Authorizes Cloud Run, Build, Artifacts, and Secrets. |
| **Compute Service Account** | Found | Identity used by Cloud Run at runtime. |
| **Secret Accessor Role** | Assigned | Grants your Go code permission to read `DB_PASSWORD`. |
| **GCP Project Context** | Set Locally | Handled via `gcloud config set project YOUR_PROJECT_ID`. |
