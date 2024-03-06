package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/bufbuild/protovalidate-go"
	"github.com/dmitrorezn/proto-validation-example/gen/apis/processor"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProcessorSvc_Process(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc, err := newProcessorSvc(ctx)
	assert.NoError(t, err)

	request := processor.ProcessRequest{
		Name: "",
	}
	_, err = svc.Process(ctx, &request)
	assert.Error(t, err)

	verr := &protovalidate.ValidationError{}
	assert.True(t, errors.As(err, &verr))
	fmt.Println("verr", verr)

	if assert.Len(t, verr.Violations, 1) {
		assert.Equal(t, verr.Violations[0].Message, "name: value length must be at least 1 characters [string.min_len]")
	}

	request = processor.ProcessRequest{
		Name: "Test",
	}
	resp, err := svc.Process(ctx, &request)
	assert.NoError(t, err)

	assert.NotEmpty(t, resp.GetId())
}
