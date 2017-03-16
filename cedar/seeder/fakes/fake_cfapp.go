// This file was generated by counterfeiter
package fakes

import (
	"sync"
	"time"

	"code.cloudfoundry.org/diego-stress-tests/cedar/cli"
	"code.cloudfoundry.org/diego-stress-tests/cedar/seeder"
	"code.cloudfoundry.org/lager"
	"golang.org/x/net/context"
)

type FakeCfApp struct {
	AppNameStub        func() string
	appNameMutex       sync.RWMutex
	appNameArgsForCall []struct{}
	appNameReturns     struct {
		result1 string
	}
	appNameReturnsOnCall map[int]struct {
		result1 string
	}
	AppURLStub        func() string
	appURLMutex       sync.RWMutex
	appURLArgsForCall []struct{}
	appURLReturns     struct {
		result1 string
	}
	appURLReturnsOnCall map[int]struct {
		result1 string
	}
	PushStub        func(logger lager.Logger, ctx context.Context, client cli.CFClient, payload string, timeout time.Duration) error
	pushMutex       sync.RWMutex
	pushArgsForCall []struct {
		logger  lager.Logger
		ctx     context.Context
		client  cli.CFClient
		payload string
		timeout time.Duration
	}
	pushReturns struct {
		result1 error
	}
	pushReturnsOnCall map[int]struct {
		result1 error
	}
	StartStub        func(logger lager.Logger, ctx context.Context, client cli.CFClient, skipVerifyCertificate bool, timeout time.Duration) error
	startMutex       sync.RWMutex
	startArgsForCall []struct {
		logger                lager.Logger
		ctx                   context.Context
		client                cli.CFClient
		skipVerifyCertificate bool
		timeout               time.Duration
	}
	startReturns struct {
		result1 error
	}
	startReturnsOnCall map[int]struct {
		result1 error
	}
	GuidStub        func(logger lager.Logger, ctx context.Context, client cli.CFClient, timeout time.Duration) (string, error)
	guidMutex       sync.RWMutex
	guidArgsForCall []struct {
		logger  lager.Logger
		ctx     context.Context
		client  cli.CFClient
		timeout time.Duration
	}
	guidReturns struct {
		result1 string
		result2 error
	}
	guidReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCfApp) AppName() string {
	fake.appNameMutex.Lock()
	ret, specificReturn := fake.appNameReturnsOnCall[len(fake.appNameArgsForCall)]
	fake.appNameArgsForCall = append(fake.appNameArgsForCall, struct{}{})
	fake.recordInvocation("AppName", []interface{}{})
	fake.appNameMutex.Unlock()
	if fake.AppNameStub != nil {
		return fake.AppNameStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.appNameReturns.result1
}

func (fake *FakeCfApp) AppNameCallCount() int {
	fake.appNameMutex.RLock()
	defer fake.appNameMutex.RUnlock()
	return len(fake.appNameArgsForCall)
}

func (fake *FakeCfApp) AppNameReturns(result1 string) {
	fake.AppNameStub = nil
	fake.appNameReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeCfApp) AppNameReturnsOnCall(i int, result1 string) {
	fake.AppNameStub = nil
	if fake.appNameReturnsOnCall == nil {
		fake.appNameReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.appNameReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeCfApp) AppURL() string {
	fake.appURLMutex.Lock()
	ret, specificReturn := fake.appURLReturnsOnCall[len(fake.appURLArgsForCall)]
	fake.appURLArgsForCall = append(fake.appURLArgsForCall, struct{}{})
	fake.recordInvocation("AppURL", []interface{}{})
	fake.appURLMutex.Unlock()
	if fake.AppURLStub != nil {
		return fake.AppURLStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.appURLReturns.result1
}

func (fake *FakeCfApp) AppURLCallCount() int {
	fake.appURLMutex.RLock()
	defer fake.appURLMutex.RUnlock()
	return len(fake.appURLArgsForCall)
}

func (fake *FakeCfApp) AppURLReturns(result1 string) {
	fake.AppURLStub = nil
	fake.appURLReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeCfApp) AppURLReturnsOnCall(i int, result1 string) {
	fake.AppURLStub = nil
	if fake.appURLReturnsOnCall == nil {
		fake.appURLReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.appURLReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *FakeCfApp) Push(logger lager.Logger, ctx context.Context, client cli.CFClient, payload string, timeout time.Duration) error {
	fake.pushMutex.Lock()
	ret, specificReturn := fake.pushReturnsOnCall[len(fake.pushArgsForCall)]
	fake.pushArgsForCall = append(fake.pushArgsForCall, struct {
		logger  lager.Logger
		ctx     context.Context
		client  cli.CFClient
		payload string
		timeout time.Duration
	}{logger, ctx, client, payload, timeout})
	fake.recordInvocation("Push", []interface{}{logger, ctx, client, payload, timeout})
	fake.pushMutex.Unlock()
	if fake.PushStub != nil {
		return fake.PushStub(logger, ctx, client, payload, timeout)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.pushReturns.result1
}

func (fake *FakeCfApp) PushCallCount() int {
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	return len(fake.pushArgsForCall)
}

func (fake *FakeCfApp) PushArgsForCall(i int) (lager.Logger, context.Context, cli.CFClient, string, time.Duration) {
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	return fake.pushArgsForCall[i].logger, fake.pushArgsForCall[i].ctx, fake.pushArgsForCall[i].client, fake.pushArgsForCall[i].payload, fake.pushArgsForCall[i].timeout
}

func (fake *FakeCfApp) PushReturns(result1 error) {
	fake.PushStub = nil
	fake.pushReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCfApp) PushReturnsOnCall(i int, result1 error) {
	fake.PushStub = nil
	if fake.pushReturnsOnCall == nil {
		fake.pushReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.pushReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCfApp) Start(logger lager.Logger, ctx context.Context, client cli.CFClient, skipVerifyCertificate bool, timeout time.Duration) error {
	fake.startMutex.Lock()
	ret, specificReturn := fake.startReturnsOnCall[len(fake.startArgsForCall)]
	fake.startArgsForCall = append(fake.startArgsForCall, struct {
		logger                lager.Logger
		ctx                   context.Context
		client                cli.CFClient
		skipVerifyCertificate bool
		timeout               time.Duration
	}{logger, ctx, client, skipVerifyCertificate, timeout})
	fake.recordInvocation("Start", []interface{}{logger, ctx, client, skipVerifyCertificate, timeout})
	fake.startMutex.Unlock()
	if fake.StartStub != nil {
		return fake.StartStub(logger, ctx, client, skipVerifyCertificate, timeout)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.startReturns.result1
}

func (fake *FakeCfApp) StartCallCount() int {
	fake.startMutex.RLock()
	defer fake.startMutex.RUnlock()
	return len(fake.startArgsForCall)
}

func (fake *FakeCfApp) StartArgsForCall(i int) (lager.Logger, context.Context, cli.CFClient, bool, time.Duration) {
	fake.startMutex.RLock()
	defer fake.startMutex.RUnlock()
	return fake.startArgsForCall[i].logger, fake.startArgsForCall[i].ctx, fake.startArgsForCall[i].client, fake.startArgsForCall[i].skipVerifyCertificate, fake.startArgsForCall[i].timeout
}

func (fake *FakeCfApp) StartReturns(result1 error) {
	fake.StartStub = nil
	fake.startReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCfApp) StartReturnsOnCall(i int, result1 error) {
	fake.StartStub = nil
	if fake.startReturnsOnCall == nil {
		fake.startReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.startReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCfApp) Guid(logger lager.Logger, ctx context.Context, client cli.CFClient, timeout time.Duration) (string, error) {
	fake.guidMutex.Lock()
	ret, specificReturn := fake.guidReturnsOnCall[len(fake.guidArgsForCall)]
	fake.guidArgsForCall = append(fake.guidArgsForCall, struct {
		logger  lager.Logger
		ctx     context.Context
		client  cli.CFClient
		timeout time.Duration
	}{logger, ctx, client, timeout})
	fake.recordInvocation("Guid", []interface{}{logger, ctx, client, timeout})
	fake.guidMutex.Unlock()
	if fake.GuidStub != nil {
		return fake.GuidStub(logger, ctx, client, timeout)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fake.guidReturns.result1, fake.guidReturns.result2
}

func (fake *FakeCfApp) GuidCallCount() int {
	fake.guidMutex.RLock()
	defer fake.guidMutex.RUnlock()
	return len(fake.guidArgsForCall)
}

func (fake *FakeCfApp) GuidArgsForCall(i int) (lager.Logger, context.Context, cli.CFClient, time.Duration) {
	fake.guidMutex.RLock()
	defer fake.guidMutex.RUnlock()
	return fake.guidArgsForCall[i].logger, fake.guidArgsForCall[i].ctx, fake.guidArgsForCall[i].client, fake.guidArgsForCall[i].timeout
}

func (fake *FakeCfApp) GuidReturns(result1 string, result2 error) {
	fake.GuidStub = nil
	fake.guidReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeCfApp) GuidReturnsOnCall(i int, result1 string, result2 error) {
	fake.GuidStub = nil
	if fake.guidReturnsOnCall == nil {
		fake.guidReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.guidReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeCfApp) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.appNameMutex.RLock()
	defer fake.appNameMutex.RUnlock()
	fake.appURLMutex.RLock()
	defer fake.appURLMutex.RUnlock()
	fake.pushMutex.RLock()
	defer fake.pushMutex.RUnlock()
	fake.startMutex.RLock()
	defer fake.startMutex.RUnlock()
	fake.guidMutex.RLock()
	defer fake.guidMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeCfApp) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ seeder.CfApp = new(FakeCfApp)
