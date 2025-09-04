package cf

import (
	"fmt"
	"os"
	"runtime"
)

const (
	ProductName = "app-autoscaler"
)

// getBuildInfo extracts version and repository information from environment variables
func getBuildInfo() (version, repoURL string) {
	// Get version from DT_RELEASE_BUILD_VERSION environment variable
	version = os.Getenv("DT_RELEASE_BUILD_VERSION")
	if version == "" {
		version = "unknown"
	}

	// Get repository URL from GO_INSTALL_PACKAGE_SPEC environment variable
	repoURL = os.Getenv("GO_INSTALL_PACKAGE_SPEC")
	if repoURL == "" {
		repoURL = "somehow related to github.com/cloudfoundry/app-autoscaler"
	}

	return
}

// GetUserAgent returns a custom HTTP User-Agent string in the format:
// app-autoscaler/{version} ({repoURL}; {version}) Go/{goVersion} {os}/{arch}
func GetUserAgent() string {
	version, repoURL := getBuildInfo()

	systemInfo := fmt.Sprintf("%s; %s", repoURL, version)
	platformInfo := fmt.Sprintf("Go/%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	return fmt.Sprintf("%s/%s (%s) %s", ProductName, version, systemInfo, platformInfo)
}
