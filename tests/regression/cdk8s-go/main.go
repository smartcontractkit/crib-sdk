package main

import (
	"fmt"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/cdk8s-team/cdk8s-core-go/cdk8s/v2"
	"os"
	"os/exec"
)

type MyChartProps struct {
	cdk8s.ChartProps
}

func NewMyChart(scope constructs.Construct, id string, props *MyChartProps) cdk8s.Chart {
	var cprops cdk8s.ChartProps
	if props != nil {
		cprops = props.ChartProps
	}
	chart := cdk8s.NewChart(scope, jsii.String(id), &cprops)

	bin, err := exec.LookPath("helm")
	if err != nil {
		panic("failed to locate helm binary: " + err.Error())
	}

	cdk8s.NewHelm(chart, jsii.String("my-helm-chart"), &cdk8s.HelmProps{
		Chart:          jsii.String("oci://registry-1.docker.io/bitnamicharts/postgresql"),
		HelmExecutable: jsii.String(bin),
		HelmFlags: &[]*string{
			jsii.String("--skip-tests"),
		},
		Namespace:   jsii.String("default"),
		ReleaseName: jsii.String("my-postgres-release"),
		Values: &map[string]any{
			"architecture":     "standalone",
			"fullnameOverride": "my-postgres-release",
			"tls": map[string]any{
				"enabled": false,
			},
			"networkPolicy": map[string]any{
				"enabled": false,
			},
			"volumePermissions": map[string]any{
				"enabled": false,
			},
			"auth": map[string]any{
				"enablePostgresUser": true,
				"postgresPassword":   "postgres",
			},
			"containerPorts": map[string]any{
				"postgresql": 5432,
			},
			"metrics": map[string]any{
				"enabled": false,
			},
			"image": map[string]any{
				"registry":   "docker.io",
				"repository": "bitnami/postgresql",
			},
			"primary": map[string]any{
				"persistence": map[string]any{
					"enabled": false,
				},
			},
			"initdb": map[string]any{
				"scripts": map[string]string{
					"init.sql": `CREATE USER chainlink_user_0 WITH PASSWORD 'chainlink_pass_0';\nCREATE DATABASE chainlink_node_0 OWNER chainlink_user_0;\nGRANT ALL PRIVILEGES ON DATABASE chainlink_node_0 TO chainlink_user_0;\nCREATE USER chainlink_user_1 WITH PASSWORD 'chainlink_pass_1';\nCREATE DATABASE chainlink_node_1 OWNER chainlink_user_1;\nGRANT ALL PRIVILEGES ON DATABASE chainlink_node_1 TO chainlink_user_1;\nCREATE USER chainlink_user_2 WITH PASSWORD 'chainlink_pass_2';\nCREATE DATABASE chainlink_node_2 OWNER chainlink_user_2;\nGRANT ALL PRIVILEGES ON DATABASE chainlink_node_2 TO chainlink_user_2;`,
				},
			},
		},
		Version: jsii.String("16.7.10"),
	})

	return chart
}

func main() {
	app := cdk8s.NewApp(nil)
	NewMyChart(app, "regression", nil)
	app.Synth()
	fmt.Fprintf(os.Stderr, *app.SynthYaml())
}
