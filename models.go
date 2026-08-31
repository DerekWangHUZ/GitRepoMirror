package main

import "time"

const (
	PlatformGitHub  = "github"
	PlatformGitLab  = "gitlab"
	PlatformGeneric = "generic"

	ManagementCLI     = "cli"
	ManagementGitOnly = "git-only"
)

type RemoteSpec struct {
	Platform    string `json:"platform"`
	Host        string `json:"host"`
	Namespace   string `json:"namespace"`
	Repository  string `json:"repository"`
	CloneURL    string `json:"cloneUrl"`
	WebURL      string `json:"webUrl,omitempty"`
	DisplayName string `json:"displayName"`
}

type Settings struct {
	Proxy       string `json:"proxy"`
	Prefix      string `json:"prefix"`
	Suffix      string `json:"suffix"`
	Concurrency int    `json:"concurrency"`
	SyncLFS     bool   `json:"syncLfs"`
}

type Repository struct {
	ID             string     `json:"id"`
	Source         RemoteSpec `json:"source"`
	Target         RemoteSpec `json:"target"`
	ManagementMode string     `json:"managementMode"`
	Visibility     string     `json:"visibility,omitempty"`
	Mode           string     `json:"mode"`
	Status         string     `json:"status"`
	LastSync       time.Time  `json:"lastSync"`
	LastError      string     `json:"lastError,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type StoreData struct {
	Settings     Settings     `json:"settings"`
	Repositories []Repository `json:"repositories"`
}

type ToolStatus struct {
	Installed     bool   `json:"installed"`
	Path          string `json:"path,omitempty"`
	Authenticated bool   `json:"authenticated"`
	AuthState     string `json:"authState,omitempty"`
	Login         string `json:"login,omitempty"`
	Message       string `json:"message,omitempty"`
}

type EnvironmentStatus struct {
	Git    ToolStatus `json:"git"`
	GitLFS ToolStatus `json:"gitLfs"`
	GitHub ToolStatus `json:"github"`
	GitLab ToolStatus `json:"gitlab"`
}

type MirrorRequest struct {
	Source          string `json:"source"`
	TargetPlatform  string `json:"targetPlatform"`
	TargetNamespace string `json:"targetNamespace"`
	TargetName      string `json:"targetName"`
	TargetURL       string `json:"targetUrl"`
	Visibility      string `json:"visibility"`
	Prefix          string `json:"prefix"`
	Suffix          string `json:"suffix"`
	Mode            string `json:"mode"`
	ConflictPolicy  string `json:"conflictPolicy"`
	SyncLFS         bool   `json:"syncLfs"`
}

type LogEvent struct {
	RepositoryID string `json:"repositoryId,omitempty"`
	Level        string `json:"level"`
	Message      string `json:"message"`
	Time         string `json:"time"`
}

type ProgressEvent struct {
	RepositoryID string `json:"repositoryId,omitempty"`
	Step         int    `json:"step"`
	Label        string `json:"label"`
	Status       string `json:"status"`
}
