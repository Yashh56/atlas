# Vercel Deployment Provider

Atlas supports deploying your front-end and full-stack applications directly to Vercel. 

## Supported Frameworks

Atlas heuristic-based project analyzer can automatically detect and deploy standard web frameworks to Vercel, including:
- React
- Next.js
- Vue
- Nuxt
- Angular
- Svelte
- Vanilla HTML/JS

## Configuration

To deploy to Vercel, you need to provide your Vercel credentials to Atlas. You can do this in two ways:

1. **Environment Variable**: Set the `VERCEL_TOKEN` environment variable in your `.env` file or shell profile. This is the recommended approach for CI/CD environments.

2. **Stored Credentials**: Run the interactive CLI to securely store a token on your local machine:
   ```bash
   atlas providers set vercel
   ```

## Usage

To trigger a deployment to Vercel, you simply pass `vercel` as the deployment provider:

```bash
atlas deploy /path/to/project --provider vercel
```

### Authentication Flow
When you run a deployment, Atlas will check for authentication in the following order:
1. `VERCEL_TOKEN` environment variable.
2. Locally stored credentials (`atlas providers set vercel`).
3. CLI-delegated Authentication: If you have the Vercel CLI installed and are already authenticated, Atlas will piggyback on your existing Vercel session (`vercel whoami`).
4. Interactive Login: If no credentials are found, Atlas will interactively prompt you to login by running `vercel login` internally.

## Git Preconditions
Unlike some other deployment platforms, Vercel deployments via Atlas use the Vercel CLI under the hood and **do not require your Git working tree to be clean or pushed to a remote repository**. You can freely deploy dirty working directories and unpushed commits.
