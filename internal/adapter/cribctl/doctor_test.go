package cribctl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_extractVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc  string
		input string
		want  string
	}{
		{
			desc: "kubectl", // kubectl version
			input: `Client Version: v1.30.7
Kustomize Version: v5.0.4-0.20230601165947-6ce0bf390ce3
The connection to the server localhost:8080 was refused - did you specify the right host or port?`,
			want: "1.30.7",
		},
		{
			desc:  "helm", // helm version --template="{{.Version}}"
			input: `v3.17.3`,
			want:  "3.17.3",
		},
		{
			desc:  "helm - alternative", // helm version
			input: `version.BuildInfo{Version:"v3.17.3", GitCommit:"e4da49785aa6e6ee2b86efd5dd9e43400318262b", GitTreeState:"clean", GoVersion:"go1.23.7"}`,
			want:  "3.17.3",
		},
		{
			desc:  "task", // task --version
			input: `Task version: v3.42.1 (h1:HOaFbZGLOrAy2V/dLsX2rGJZVG2Qx6268KUIAIXdNE4=)`,
			want:  "3.42.1",
		},
		{
			desc:  "kind", // kind --version
			input: `kind version 0.27.0`,
			want:  "0.27.0",
		},
		{
			desc:  "telepresence", // telepresence version
			input: `telepresence version 2.10.0 (api v3)`,
			want:  "2.10.0",
		},
		{
			desc:  "node", // node -v
			input: `v22.17.0`,
			want:  "22.17.0",
		},
		{
			desc:  "asdf", // asdf --version
			input: `v0.14.1-f00f759`,
			want:  "0.14.1",
		},
		{
			desc:  "go", // go version
			input: `go version go1.24.3 darwin/arm64`,
			want:  "1.24.3",
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got, err := (&dependency{}).extractVersion([]byte(tc.input))
			assert.NoError(t, err, "extractVersion should not return an error")
			assert.Equal(t, tc.want, got, "extractVersion should return the expected version")
		})
	}
}
