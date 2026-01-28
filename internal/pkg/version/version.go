package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

var (
	Version   = "dev"
	BuildTime = ""
	GoVersion = runtime.Version()
	GitCommit = ""
)

func GetLatestPrismVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Yat-Muk/prism-v2/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", nil
	}
	return rel.TagName, nil
}

func Short() string {
	// v1.0.0-abcdef01 這種格式
	if GitCommit != "" {
		return fmt.Sprintf("v%s (%s)", Version, GitCommit)
	}
	return "v" + Version
}

func Info() string {
	return fmt.Sprintf(
		"Prism v%s\nBuild Time: %s\nGo Version: %s\nGit Commit: %s",
		Version, BuildTime, GoVersion, GitCommit,
	)
}
