package installer

import (
	"fmt"
	"os"
	"strings"
)

type EnvVar struct {
	Name     string
	Secret   bool
	Optional bool
	Default  string
}

var envContract = []EnvVar{
	{Name: "git_username"},
	{Name: "git_user_path"},
	{Name: "github_pat", Secret: true},
	{Name: "system_root_app_repo"},
	{Name: "system_root_app_path"},
	{Name: "cloudflare_token", Secret: true},
	{Name: "vps"},
	{Name: "base_domain"},
	{Name: "harbor_hostname"},
	{Name: "harbor_initial_password", Secret: true},
	{Name: "harbor_chart_version"},
	{Name: "traefik_dashboard"},
	{Name: "traefik_email"},
	{Name: "traefik_chart_version"},
	{Name: "jenkins_hostname"},
	{Name: "jenkins_initial_password", Secret: true},
	{Name: "jenkins_chart_version"},
	{Name: "jenkins_pipeline_library_repo"},
	{Name: "jenkins_pipeline_library_path"},
	{Name: "jenkins_github_org_folder_name", Optional: true, Default: "Repositories"},
	{Name: "jenkins_github_org_folder_repo_filter", Optional: true, Default: "*"},
	{Name: "jenkins_jenkinsfile_path", Optional: true, Default: "infrastructure/Jenkinsfile"},
}

type Config struct {
	Values map[string]string
}

func LoadConfig() (Config, error) {
	values := make(map[string]string, len(envContract))
	var missing []string

	for _, item := range envContract {
		value := os.Getenv(item.Name)
		if value == "" && item.Optional {
			value = item.Default
		}
		if value == "" && !item.Optional {
			missing = append(missing, item.Name)
			continue
		}
		values[item.Name] = value
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return Config{Values: values}, nil
}

func RedactedValue(name, value string) string {
	for _, item := range envContract {
		if item.Name == name && item.Secret {
			if value == "" {
				return ""
			}
			return "[set]"
		}
	}
	return value
}
