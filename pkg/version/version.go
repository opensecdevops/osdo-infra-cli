package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the CLI
	Version = "0.1.0"
	
	// BuildDate is when the binary was built
	BuildDate = "unknown"
	
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	
	// GitBranch is the git branch
	GitBranch = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
	GitBranch string `json:"gitBranch"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("OSDO Infrastructure CLI\nVersion: %s\nBuild Date: %s\nGit Commit: %s\nGit Branch: %s\nGo Version: %s\nCompiler: %s\nPlatform: %s",
		i.Version, i.BuildDate, i.GitCommit, i.GitBranch, i.GoVersion, i.Compiler, i.Platform)
}
