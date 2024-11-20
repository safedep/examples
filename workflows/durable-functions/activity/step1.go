package activity

import (
	"context"
	"fmt"
)

func Step1Activity(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Step1Activity: %s", input), nil
}
