# Render Deployment Provider

Atlas supports triggering deployments for web services and static sites hosted on [Render](https://render.com/).

## Pre-requisites

Render deployments through Atlas operate differently from Vercel deployments. Atlas **does not** create Render services for you from scratch. You must first create and configure your Web Service or Static Site in the Render dashboard. Once the service exists, Atlas can orchestrate, trigger, and monitor subsequent deployments.

## Configuration

To deploy to Render, you need two pieces of configuration:

### 1. Authentication
You need to provide your personal Render API key. You can do this in two ways:

- **Environment Variable**: Set the `RENDER_TOKEN` environment variable in your `.env` file or shell profile. This is the recommended approach for CI/CD environments.
- **Stored Credentials**: Run the interactive CLI to securely store your token locally:
  ```bash
  atlas providers set render
  ```

### 2. Service ID Configuration
You must link your local project to a specific Render Service ID (e.g., `srv-c...`). You can obtain this ID from the Render dashboard (in your service's URL or settings).

To configure the Service ID for your project, run:
```bash
atlas providers set render --service-id srv-your-service-id
```
This stores the `render_service_id` safely in your local `.atlas/project.json` configuration.

## Usage

To trigger a deployment, use:

```bash
atlas deploy /path/to/project --provider render
```

### Git Push Preconditions
Because Render builds and deploys your code directly from the connected remote Git repository (e.g., GitHub, GitLab), **Atlas enforces a strict push-check for Render deployments**.

Before triggering a deployment, Atlas will check that:
1. Your working directory is clean.
2. The current commit (`HEAD`) on your local machine exactly matches the commit on the remote tracking branch (e.g., `origin/main`).

If you have uncommitted changes or unpushed commits, the deployment will fail fast, prompting you to push your changes first.

### Deployment Orchestration
When you run a deployment, Atlas performs the following steps:
1. **Validates** authentication and the Service ID.
2. **Checks** your Git tree to ensure all code is pushed to the remote repository.
3. **Triggers** a new deployment using the Render API, pinning the deployment to the exact Git commit SHA that you are currently on.
4. **Polls** the Render API to monitor the deployment status in real-time.
5. **Returns** the live URL of your service once the deployment succeeds (or reports a failure if the build crashes).
