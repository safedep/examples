package main

import (
	"context"
	"fmt"
	"time"

	"github.com/safedep/examples/workflow/durable-functions/workflow"
	"go.temporal.io/sdk/client"
)

func main() {
	tc, err := client.Dial(client.Options{})
	if err != nil {
		panic(err)
	}

	defer tc.Close()

	input := workflow.WorkflowInput{
		Val: "Input Value",
	}

	we, err := tc.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        fmt.Sprintf("SimpleAnalysisWorkflow_%v", time.Now().Unix()),
		TaskQueue: workflow.SimpleWorkflowTaskQueue,
	}, workflow.SimpleAnalysisWorkflow, input)
	if err != nil {
		panic(err)
	}

	var result workflow.WorkflowOutput
	err = we.Get(context.Background(), &result)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Result: %+v\n", result)
}
