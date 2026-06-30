// Command axisml-scheduler is the AxisML batch scheduler: a scheduler-framework
// binary that composes upstream scheduler-plugins Coscheduling (gang) and the
// in-tree NodeResourcesFit scoring (binpack, configured via
// KubeSchedulerConfiguration) with the AxisML ElasticScheduling plugin
// (label-bound ElasticQuota enforcement).
package main

import (
	"os"

	"k8s.io/component-base/cli"
	_ "k8s.io/component-base/metrics/prometheus/clientgo" // register client-go metrics
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
	"sigs.k8s.io/scheduler-plugins/pkg/coscheduling"

	"github.com/axisml/axisml/axisml-infra/axisml-scheduler/internal/plugins/elasticscheduling"
)

func main() {
	cmd := app.NewSchedulerCommand(
		app.WithPlugin(coscheduling.Name, coscheduling.New),
		app.WithPlugin(elasticscheduling.Name, elasticscheduling.New),
	)
	os.Exit(cli.Run(cmd))
}
