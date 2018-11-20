// Copyright 2018 Authors of Cilium
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/cilium/cilium/pkg/endpointmanager"
	"github.com/cilium/cilium/pkg/option"
	"github.com/cilium/cilium/pkg/pidfile"
)

var (
	// cleanUPSig channel that is closed when the daemon agent should be
	// terminated.
	cleanUPSig = make(chan struct{})
	// cleanUPWg all cleanup operations will be marked as Done() when completed.
	cleanUPWg = &sync.WaitGroup{}
)

func handleInterrupt() <-chan struct{} {
	// Handle the handleOSSignals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	interrupt := make(chan struct{})
	go func() {
		for s := range sig {
			log.WithField("signal", s).Info("Exiting due to signal")
			<-pidfile.Clean()
			<-Clean()
			if option.Config.PolicyEnforcementCleanUp {
				for _, ep := range endpointmanager.GetEndpoints() {
					ep.DeleteBPFProgramLocked()
				}
			}
			break
		}
		close(interrupt)
	}()
	return interrupt
}

// Clean cleans up everything created by this package. It closes the returned
// channel once everything is cleaned up.
func Clean() <-chan struct{} {
	close(cleanUPSig)
	exited := make(chan struct{})
	go func() {
		cleanUPWg.Wait()
		close(exited)
	}()
	return exited
}
