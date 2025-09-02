package handler

import (
	"context"

	"blueprint/config"
	"blueprint/pkg/logger"
	pb "blueprint/proto/blueprint"
	"testing"

	. "github.com/modern-go/test"
	. "github.com/modern-go/test/must"
	 
)

// Unit test all your gPRC calls and make sure all param correct and results also valid
func TestCall(t *testing.T) {

	// get instant of handler

	// config instant
	cfg := config.NewConfig()	
	t.Run("Config should have value ", Case(func(ctx context.Context) {
		Assert(cfg !=nil)

	}))

	
		// pass to logger cobfiurations

	log, _ := logger.NewLogger(cfg)

	h := Blueprint{
		Log: log,
	}

	req := pb.CallRequest{
		Name: "value",
	}

	rsp, _ := h.Call(context.Background(), &req)
	if rsp.Msg != "Hello value" {
		t.Fatalf("%v", rsp)
	}

}
