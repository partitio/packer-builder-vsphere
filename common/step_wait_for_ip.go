package common

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer/helper/multistep"
	"github.com/hashicorp/packer/packer"
	"github.com/jetbrains-infra/packer-builder-vsphere/driver"
)

type StepWaitForIp struct{}

func (s *StepWaitForIp) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	vm := state.Get("vm").(*driver.VirtualMachine)

	ui.Say("Waiting for IP...")
	retries := 5
	ipChan := make(chan string)
	errChan := make(chan error)
	go func() {
		var err error
		var ip string
		for i := 1; i <= retries; i++ {
			ip, err = vm.WaitForIP(ctx)
			if err == nil {
				ipChan <- ip
				break
			}
			if i != retries {
				ui.Say(fmt.Sprintf("Failed to get VM IP: %s", err))
				ui.Say(fmt.Sprintf("Retrying to get VM IP... (Remaining attempts: %d)", retries-i))
			}
		}
		if err != nil {
			errChan <- err
		}
	}()

	for {
		select {
		case err := <-errChan:
			state.Put("error", err)
			return multistep.ActionHalt
		case <-ctx.Done():
			return multistep.ActionHalt
		case ip := <-ipChan:
			state.Put("ip", ip)
			ui.Say(fmt.Sprintf("IP address: %v", ip))
			return multistep.ActionContinue
		case <-time.After(1 * time.Second):
			if _, ok := state.GetOk(multistep.StateCancelled); ok {
				return multistep.ActionHalt
			}
		}
	}
}

func (s *StepWaitForIp) Cleanup(state multistep.StateBag) {}
