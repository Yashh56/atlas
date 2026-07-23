# Netlify Deployment Provider

Atlas supports deploying your front-end and full-stack applications directly to Netlify. 

## Supported Frameworks

Atlas automatically detects and deploys standard web frameworks to Netlify, including:
- React (Create React App, Vite)
- Next.js (Requires static export)
- Vue
- Nuxt
- Angular
- Svelte
- Vanilla HTML/JS

*Note: Netlify only serves static output. Go projects or other back-end API frameworks will return an error during the deployment phase if attempted.*

## Configuration

To deploy to Netlify, you need to provide your Netlify credentials to Atlas. You can do this in two ways:

1. **Environment Variable**: Set the `NETLIFY_TOKEN` environment variable in your `.env` file or shell profile. This is the recommended approach for CI/CD environments.

2. **Stored Credentials**: Run the interactive CLI to securely store a token on your local machine:
   ```bash
   atlas providers set netlify
   ```

## Usage

To trigger a deployment to Netlify, you simply pass `netlify` as the deployment provider:

```bash
atlas /path/to/project --action deploy --provider netlify
```

### Manual Output Directory Override
Atlas uses heuristics to determine your project's publish directory (e.g. `dist`, `build`, `out`). If your project uses a custom output directory, you can override the auto-detection by passing the `--output-dir` flag:
```bash
atlas /path/to/project --action deploy --provider netlify --output-dir my-custom-build-folder
```

### Authentication Flow
When you run a deployment, Atlas will check for authentication in the following order:
1. `NETLIFY_TOKEN` environment variable.
2. Locally stored credentials (`atlas providers set netlify`).
3. CLI-delegated Authentication: If you have the Netlify CLI installed and are already authenticated, Atlas will piggyback on your existing Netlify session (`netlify api getCurrentUser`).
4. Interactive Login: If no credentials are found, Atlas will interactively prompt you to login by running `netlify login` internally.

## Site Resolution

Atlas auto-resolves your Netlify site for deployment:
1. It looks for a `netlify_site_id` in your project's `.atlas/project.json`.
2. If none exists, Atlas will automatically create a new Netlify site for your workspace using `netlify sites:create`.
3. If the workspace name is already taken, Atlas handles the collision by generating a unique suffix and retrying.
4. Atlas saves the resolved Site ID in `.atlas/project.json` for future deployments.
