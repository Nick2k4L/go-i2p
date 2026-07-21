setup:
	GOEXPERIMENT=jsonv2 go install github.com/aquasecurity/trivy/cmd/trivy@latest
	go install github.com/leaanthony/comply@latest
	go install mvdan.cc/gofumpt@latest
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest # update to a stable version of gomarkdoc
	go install github.com/github-release/github-release@latest
	go install github.com/ofabry/go-callvis@latest
	go install github.com/evilmartians/lefthook@latest
	lefthook install
