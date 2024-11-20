package activity

import (
	"context"
	"fmt"
)

func Step2Activity(ctx context.Context, input string) (string, error) {
	return fmt.Sprintf("Step2Activity: %s", input), nil
}
