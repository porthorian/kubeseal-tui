package app

type DataSource string

const (
	DataSourceValue DataSource = "value"
	DataSourceFile  DataSource = "file"
)

type DataEntry struct {
	Key     string
	Source  DataSource
	Payload string
}

type Target struct {
	Namespace string
	Dir       string
	File      string
}

type Config struct {
	SecretName          string
	SecretType          string
	ControllerNamespace string
	ControllerName      string
	Force               bool
	CWD                 string
	Data                []DataEntry
	Targets             []Target
}
