package workflow

import (
	"errors"
	"time"

	"github.com/safedep/examples/workflow/durable-functions/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	SimpleWorkflowTaskQueue = "simple-workflow-tq"
)

var (
	ErrNoRetry = errors.New("no retry")
)

type WorkflowInput struct {
	Val string
}

type WorkflowOutput struct {
	Values []string
}

func SimpleAnalysisWorkflow(ctx workflow.Context, input WorkflowInput) (WorkflowOutput, error) {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     2.0,
			MaximumInterval:        time.Minute,
			MaximumAttempts:        5,
			NonRetryableErrorTypes: []string{ErrNoRetry.Error()},
		},
	}

	ctx = workflow.WithActivityOptions(ctx, options)

	var output WorkflowOutput
	var stepOutput string

	// Compromises type safety of Activity arguments by using interface{}
	err := workflow.ExecuteActivity(ctx, activity.Step1Activity, input.Val).Get(ctx, &stepOutput)
	if err != nil {
		return WorkflowOutput{}, err
	}

	output.Values = append(output.Values, stepOutput)

	err = workflow.ExecuteActivity(ctx, activity.Step2Activity, input.Val).Get(ctx, &stepOutput)
	if err != nil {
		return WorkflowOutput{}, err
	}

	output.Values = append(output.Values, stepOutput)

	return output, nil
}
