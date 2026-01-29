module github.com/open-cli-collective/jira-ticket-cli

go 1.24.0

require (
	github.com/fatih/color v1.18.0
	github.com/open-cli-collective/atlassian-go v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/yuin/goldmark v1.7.16
)

replace github.com/open-cli-collective/atlassian-go => ../../shared

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.38.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
