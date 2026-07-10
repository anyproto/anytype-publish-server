package publish

type configGetter interface {
	GetPublish() Config
}

const (
	DefaultMembershipLimit       = int64(6000 << 20)
	DefaultDefaultLimit          = int64(10 << 20)
	DefaultBaseUrlTemplate       = "https://any.coop/%s"
	DefaultInviteLinkUrlTemplate = "https://invite.any.coop/%s#%s"
	DefaultMemberUrlTemplate     = "https://%s.org"
	DefaultIndexFileName         = "index.json.gz"
)

type Config struct {
	UploadUrlPrefix       string `yaml:"uploadUrlPrefix"`
	HttpApiAddr           string `yaml:"httpApiAddr"`
	CleanupOn             bool   `yaml:"cleanupOn"`
	MembershipLimit       int64  `yaml:"membershipLimit"`
	DefaultLimit          int64  `yaml:"defaultLimit"`
	BaseUrlTemplate       string `yaml:"baseUrlTemplate"`
	InviteLinkUrlTemplate string `yaml:"inviteLinkUrlTemplate"`
	MemberUrlTemplate     string `yaml:"memberUrlTemplate"`
	IndexFileName         string `yaml:"indexFileName"`
}
