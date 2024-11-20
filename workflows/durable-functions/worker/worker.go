package main

import (
	"github.com/safedep/examples/workflow/durable-functions/activity"
	"github.com/safedep/examples/workflow/durable-functions/workflow"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	tc, err := client.Dial(client.Options{})
	if err != nil {
		panic(err)
	}

	defer tc.Close()

	w := worker.New(tc, workflow.SimpleWorkflowTaskQueue, worker.Options{})

	w.RegisterWorkflow(workflow.SimpleAnalysisWorkflow)
	w.RegisterActivity(activity.Step1Activity)
	w.RegisterActivity(activity.Step2Activity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		panic(err)
	}
}
