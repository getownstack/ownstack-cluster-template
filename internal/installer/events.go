package installer

import (
	"fmt"
	"sort"
	"strings"
)

var emittedStages = map[string]bool{}

func emitStage(stage Stage) {
	if emittedStages[stage.ID] {
		return
	}
	emittedStages[stage.ID] = true
	emitEvent("stage_started", map[string]string{
		"id":    stage.ID,
		"title": stage.Title,
	})
}

func emitEvent(name string, fields map[string]string) {
	var keys []string
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{"ownstack.event=" + name}
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%q", key, fields[key]))
	}
	fmt.Println(strings.Join(parts, " "))
}

func stageFromLine(line string) {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "configuring dns record"):
		emitStage(Stages[1])
	case strings.Contains(lower, "install k3s") ||
		strings.Contains(lower, "get.k3s.io") ||
		strings.Contains(lower, "docker") ||
		strings.Contains(lower, "helmfile_") ||
		strings.Contains(lower, "yq_linux"):
		emitStage(Stages[2])
	case strings.Contains(lower, "helmfile sync") ||
		strings.Contains(lower, "cloudflare-token") ||
		strings.Contains(lower, "github-pat"):
		emitStage(Stages[3])
	case strings.Contains(lower, "harbor") ||
		strings.Contains(lower, "jenkins") ||
		strings.Contains(lower, "namespace dev") ||
		strings.Contains(lower, "namespace qa") ||
		strings.Contains(lower, "namespace prod"):
		emitStage(Stages[4])
	}
}
