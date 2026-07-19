# Render Deployment Provider

Atlas supports triggering deployments for web services and static sites hosted on [Render](https://render.com/).

## Pre-requisites

Atlas can deploy your project to Render either by linking to an existing service or by automatically creating a new service for you directly from the CLI.

## Configuration

To deploy to Render, you need two pieces of configuration:

### 1. Authentication
You need to provide your personal Render API key. You can do this in two ways:

- **Environment Variable**: Set the `RENDER_TOKEN` environment variable in your `.env` file or shell profile. This is the recommended approach for CI/CD environments.
- **Stored Credentials**: Run the interactive CLI to securely store your token locally:
  ```bash
  atlas providers set render
  ```

### 2. Service Creation & Configuration
You do **NOT** need to manually provide a Service ID or create the service on the dashboard. When you deploy a project for the first time, Atlas will:
1. Automatically detect if your project is a Static Site (e.g. React/Vite) or a Web Service (e.g. Node/Express).
2. Determine your build and start commands by analyzing your `package.json`.
3. Use the Render API to automatically create the service on your account.
4. Save the newly generated `render_service_id` safely in your local `.atlas/project.json` configuration so future deploys reuse the same service.

## Usage

To trigger a deployment, use:

```bash
atlas /path/to/project --action deploy --provider render
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
