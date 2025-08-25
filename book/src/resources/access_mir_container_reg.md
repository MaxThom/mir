# Access Mir Container Registry

The Mir container images are hosted on GitHub Container Registry (ghcr.io). This guide explains how to authenticate and pull Mir images.

## Prerequisites

  - Docker
  - Invited in the [Mir Repository](https://github.com/MaxThom/mir)

## Authentication Required

The Mir container images are hosted in a private GitHub Container Registry. You must authenticate to pull images:

```bash
# Authentication is required before pulling
docker login ghcr.io

# Then pull the image
docker pull ghcr.io/maxthom/mir:latest
```

## Creating a GitHub Personal Access Token

### Step 1: Navigate to GitHub Settings

1. Log in to your GitHub account
2. Click your profile picture in the top-right corner
3. Select **Settings** from the dropdown menu

### Step 2: Access Developer Settings

1. Scroll down to the bottom of the left sidebar
2. Click **Developer settings**

### Step 3: Create Personal Access Token

1. Click **Personal access tokens** → **Tokens (classic)**
2. Click **Generate new token** → **Generate new token (classic)**
3. Give your token a descriptive name (e.g., "Mir Container Registry")
4. Set an expiration date (or select "No expiration" for permanent tokens)
5. Select the following scopes:
   - `read:packages` - Download packages from GitHub Package Registry

6. Click **Generate token**
7. **Important**: Copy your token immediately. You won't be able to see it again!

### Alternative: Fine-grained Personal Access Token

For enhanced security, use a fine-grained token:

1. Click **Personal access tokens** → **Fine-grained tokens**
2. Click **Generate new token**
3. Set token name and expiration
4. Under **Repository access**, select:
   - "Selected repositories" and choose `maxthom/mir`
   - Or "All repositories" if you need broader access
5. Under **Permissions** → **Repository permissions**:
   - Set **Packages** to "Read" (or "Write" if needed)
6. Click **Generate token**

## Container Registry Login

### Using Personal Access Token

```bash
# Set your GitHub username and token
export GITHUB_USER="your-github-username"
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxx"

# Login to GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# Interactive login:
docker login ghcr.io
# Username: your-github-username
# Password: your-personal-access-token
```

### Verify Authentication

```bash
# Test authentication by pulling an image
docker pull ghcr.io/maxthom/mir:latest
```

## Kubernetes Secret for Image Pull

### Create Secret for Kubernetes

```bash
# Create namespace if needed
kubectl create namespace mir

# Create docker-registry secret
kubectl create secret docker-registry ghcr-mir-secret \
  --docker-server=ghcr.io \
  --docker-username=$GITHUB_USER \
  --docker-password=$GITHUB_TOKEN \
  --docker-email=your-email@example.com \
  --namespace=mir
```

### Use in Deployment

Add to your Helm values file:

```yaml
imagePullSecrets:
  - name: ghcr-mir-secret
```

## Additional Resources

- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [GitHub Personal Access Tokens](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token)
- [Docker Login Documentation](https://docs.docker.com/engine/reference/commandline/login/)
- [Kubernetes Image Pull Secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
