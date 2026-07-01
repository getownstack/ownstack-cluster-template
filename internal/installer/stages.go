package installer

type Stage struct {
	ID          string
	Title       string
	Description string
}

var Stages = []Stage{
	{
		ID:          "credentials",
		Title:       "Credentials",
		Description: "Validate environment variables and prepare access to GitHub, Cloudflare, and the VPS.",
	},
	{
		ID:          "dns",
		Title:       "DNS",
		Description: "Point the base domain and wildcard A record at the VPS with Cloudflare proxy disabled.",
	},
	{
		ID:          "system",
		Title:       "System packages",
		Description: "Install k3s, Docker, Helm, Helmfile, yq, jq, git, and supporting packages.",
	},
	{
		ID:          "platform",
		Title:       "Platform install",
		Description: "Apply the system Helmfile for Traefik, Harbor, Jenkins, and cluster RBAC.",
	},
	{
		ID:          "integration",
		Title:       "Service integration",
		Description: "Create Kubernetes secrets, Harbor robot credentials, namespaces, and Jenkins organization folders.",
	},
}
