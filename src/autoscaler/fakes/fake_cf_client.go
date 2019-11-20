// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"autoscaler/cf"
	"autoscaler/models"
	"sync"
)

type FakeCFClient struct {
	GetAppStub        func(string) (*models.AppEntity, error)
	getAppMutex       sync.RWMutex
	getAppArgsForCall []struct {
		arg1 string
	}
	getAppReturns struct {
		result1 *models.AppEntity
		result2 error
	}
	getAppReturnsOnCall map[int]struct {
		result1 *models.AppEntity
		result2 error
	}
	GetEndpointsStub        func() cf.Endpoints
	getEndpointsMutex       sync.RWMutex
	getEndpointsArgsForCall []struct {
	}
	getEndpointsReturns struct {
		result1 cf.Endpoints
	}
	getEndpointsReturnsOnCall map[int]struct {
		result1 cf.Endpoints
	}
	GetServiceInstancesInOrgStub        func(string, string) (int, error)
	getServiceInstancesInOrgMutex       sync.RWMutex
	getServiceInstancesInOrgArgsForCall []struct {
		arg1 string
		arg2 string
	}
	getServiceInstancesInOrgReturns struct {
		result1 int
		result2 error
	}
	getServiceInstancesInOrgReturnsOnCall map[int]struct {
		result1 int
		result2 error
	}
	GetTokensStub        func() cf.Tokens
	getTokensMutex       sync.RWMutex
	getTokensArgsForCall []struct {
	}
	getTokensReturns struct {
		result1 cf.Tokens
	}
	getTokensReturnsOnCall map[int]struct {
		result1 cf.Tokens
	}
	IsTokenAuthorizedStub        func(string, string) (bool, error)
	isTokenAuthorizedMutex       sync.RWMutex
	isTokenAuthorizedArgsForCall []struct {
		arg1 string
		arg2 string
	}
	isTokenAuthorizedReturns struct {
		result1 bool
		result2 error
	}
	isTokenAuthorizedReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	IsUserAdminStub        func(string) (bool, error)
	isUserAdminMutex       sync.RWMutex
	isUserAdminArgsForCall []struct {
		arg1 string
	}
	isUserAdminReturns struct {
		result1 bool
		result2 error
	}
	isUserAdminReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	IsUserSpaceDeveloperStub        func(string, string) (bool, error)
	isUserSpaceDeveloperMutex       sync.RWMutex
	isUserSpaceDeveloperArgsForCall []struct {
		arg1 string
		arg2 string
	}
	isUserSpaceDeveloperReturns struct {
		result1 bool
		result2 error
	}
	isUserSpaceDeveloperReturnsOnCall map[int]struct {
		result1 bool
		result2 error
	}
	LoginStub        func() error
	loginMutex       sync.RWMutex
	loginArgsForCall []struct {
	}
	loginReturns struct {
		result1 error
	}
	loginReturnsOnCall map[int]struct {
		result1 error
	}
	RefreshAuthTokenStub        func() (string, error)
	refreshAuthTokenMutex       sync.RWMutex
	refreshAuthTokenArgsForCall []struct {
	}
	refreshAuthTokenReturns struct {
		result1 string
		result2 error
	}
	refreshAuthTokenReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	SetAppInstancesStub        func(string, int) error
	setAppInstancesMutex       sync.RWMutex
	setAppInstancesArgsForCall []struct {
		arg1 string
		arg2 int
	}
	setAppInstancesReturns struct {
		result1 error
	}
	setAppInstancesReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeCFClient) GetApp(arg1 string) (*models.AppEntity, error) {
	fake.getAppMutex.Lock()
	ret, specificReturn := fake.getAppReturnsOnCall[len(fake.getAppArgsForCall)]
	fake.getAppArgsForCall = append(fake.getAppArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.recordInvocation("GetApp", []interface{}{arg1})
	fake.getAppMutex.Unlock()
	if fake.GetAppStub != nil {
		return fake.GetAppStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getAppReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) GetAppCallCount() int {
	fake.getAppMutex.RLock()
	defer fake.getAppMutex.RUnlock()
	return len(fake.getAppArgsForCall)
}

func (fake *FakeCFClient) GetAppCalls(stub func(string) (*models.AppEntity, error)) {
	fake.getAppMutex.Lock()
	defer fake.getAppMutex.Unlock()
	fake.GetAppStub = stub
}

func (fake *FakeCFClient) GetAppArgsForCall(i int) string {
	fake.getAppMutex.RLock()
	defer fake.getAppMutex.RUnlock()
	argsForCall := fake.getAppArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCFClient) GetAppReturns(result1 *models.AppEntity, result2 error) {
	fake.getAppMutex.Lock()
	defer fake.getAppMutex.Unlock()
	fake.GetAppStub = nil
	fake.getAppReturns = struct {
		result1 *models.AppEntity
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) GetAppReturnsOnCall(i int, result1 *models.AppEntity, result2 error) {
	fake.getAppMutex.Lock()
	defer fake.getAppMutex.Unlock()
	fake.GetAppStub = nil
	if fake.getAppReturnsOnCall == nil {
		fake.getAppReturnsOnCall = make(map[int]struct {
			result1 *models.AppEntity
			result2 error
		})
	}
	fake.getAppReturnsOnCall[i] = struct {
		result1 *models.AppEntity
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) GetEndpoints() cf.Endpoints {
	fake.getEndpointsMutex.Lock()
	ret, specificReturn := fake.getEndpointsReturnsOnCall[len(fake.getEndpointsArgsForCall)]
	fake.getEndpointsArgsForCall = append(fake.getEndpointsArgsForCall, struct {
	}{})
	fake.recordInvocation("GetEndpoints", []interface{}{})
	fake.getEndpointsMutex.Unlock()
	if fake.GetEndpointsStub != nil {
		return fake.GetEndpointsStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.getEndpointsReturns
	return fakeReturns.result1
}

func (fake *FakeCFClient) GetEndpointsCallCount() int {
	fake.getEndpointsMutex.RLock()
	defer fake.getEndpointsMutex.RUnlock()
	return len(fake.getEndpointsArgsForCall)
}

func (fake *FakeCFClient) GetEndpointsCalls(stub func() cf.Endpoints) {
	fake.getEndpointsMutex.Lock()
	defer fake.getEndpointsMutex.Unlock()
	fake.GetEndpointsStub = stub
}

func (fake *FakeCFClient) GetEndpointsReturns(result1 cf.Endpoints) {
	fake.getEndpointsMutex.Lock()
	defer fake.getEndpointsMutex.Unlock()
	fake.GetEndpointsStub = nil
	fake.getEndpointsReturns = struct {
		result1 cf.Endpoints
	}{result1}
}

func (fake *FakeCFClient) GetEndpointsReturnsOnCall(i int, result1 cf.Endpoints) {
	fake.getEndpointsMutex.Lock()
	defer fake.getEndpointsMutex.Unlock()
	fake.GetEndpointsStub = nil
	if fake.getEndpointsReturnsOnCall == nil {
		fake.getEndpointsReturnsOnCall = make(map[int]struct {
			result1 cf.Endpoints
		})
	}
	fake.getEndpointsReturnsOnCall[i] = struct {
		result1 cf.Endpoints
	}{result1}
}

func (fake *FakeCFClient) GetServiceInstancesInOrg(arg1 string, arg2 string) (int, error) {
	fake.getServiceInstancesInOrgMutex.Lock()
	ret, specificReturn := fake.getServiceInstancesInOrgReturnsOnCall[len(fake.getServiceInstancesInOrgArgsForCall)]
	fake.getServiceInstancesInOrgArgsForCall = append(fake.getServiceInstancesInOrgArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("GetServiceInstancesInOrg", []interface{}{arg1, arg2})
	fake.getServiceInstancesInOrgMutex.Unlock()
	if fake.GetServiceInstancesInOrgStub != nil {
		return fake.GetServiceInstancesInOrgStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getServiceInstancesInOrgReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) GetServiceInstancesInOrgCallCount() int {
	fake.getServiceInstancesInOrgMutex.RLock()
	defer fake.getServiceInstancesInOrgMutex.RUnlock()
	return len(fake.getServiceInstancesInOrgArgsForCall)
}

func (fake *FakeCFClient) GetServiceInstancesInOrgCalls(stub func(string, string) (int, error)) {
	fake.getServiceInstancesInOrgMutex.Lock()
	defer fake.getServiceInstancesInOrgMutex.Unlock()
	fake.GetServiceInstancesInOrgStub = stub
}

func (fake *FakeCFClient) GetServiceInstancesInOrgArgsForCall(i int) (string, string) {
	fake.getServiceInstancesInOrgMutex.RLock()
	defer fake.getServiceInstancesInOrgMutex.RUnlock()
	argsForCall := fake.getServiceInstancesInOrgArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCFClient) GetServiceInstancesInOrgReturns(result1 int, result2 error) {
	fake.getServiceInstancesInOrgMutex.Lock()
	defer fake.getServiceInstancesInOrgMutex.Unlock()
	fake.GetServiceInstancesInOrgStub = nil
	fake.getServiceInstancesInOrgReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) GetServiceInstancesInOrgReturnsOnCall(i int, result1 int, result2 error) {
	fake.getServiceInstancesInOrgMutex.Lock()
	defer fake.getServiceInstancesInOrgMutex.Unlock()
	fake.GetServiceInstancesInOrgStub = nil
	if fake.getServiceInstancesInOrgReturnsOnCall == nil {
		fake.getServiceInstancesInOrgReturnsOnCall = make(map[int]struct {
			result1 int
			result2 error
		})
	}
	fake.getServiceInstancesInOrgReturnsOnCall[i] = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) GetTokens() cf.Tokens {
	fake.getTokensMutex.Lock()
	ret, specificReturn := fake.getTokensReturnsOnCall[len(fake.getTokensArgsForCall)]
	fake.getTokensArgsForCall = append(fake.getTokensArgsForCall, struct {
	}{})
	fake.recordInvocation("GetTokens", []interface{}{})
	fake.getTokensMutex.Unlock()
	if fake.GetTokensStub != nil {
		return fake.GetTokensStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.getTokensReturns
	return fakeReturns.result1
}

func (fake *FakeCFClient) GetTokensCallCount() int {
	fake.getTokensMutex.RLock()
	defer fake.getTokensMutex.RUnlock()
	return len(fake.getTokensArgsForCall)
}

func (fake *FakeCFClient) GetTokensCalls(stub func() cf.Tokens) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = stub
}

func (fake *FakeCFClient) GetTokensReturns(result1 cf.Tokens) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = nil
	fake.getTokensReturns = struct {
		result1 cf.Tokens
	}{result1}
}

func (fake *FakeCFClient) GetTokensReturnsOnCall(i int, result1 cf.Tokens) {
	fake.getTokensMutex.Lock()
	defer fake.getTokensMutex.Unlock()
	fake.GetTokensStub = nil
	if fake.getTokensReturnsOnCall == nil {
		fake.getTokensReturnsOnCall = make(map[int]struct {
			result1 cf.Tokens
		})
	}
	fake.getTokensReturnsOnCall[i] = struct {
		result1 cf.Tokens
	}{result1}
}

func (fake *FakeCFClient) IsTokenAuthorized(arg1 string, arg2 string) (bool, error) {
	fake.isTokenAuthorizedMutex.Lock()
	ret, specificReturn := fake.isTokenAuthorizedReturnsOnCall[len(fake.isTokenAuthorizedArgsForCall)]
	fake.isTokenAuthorizedArgsForCall = append(fake.isTokenAuthorizedArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("IsTokenAuthorized", []interface{}{arg1, arg2})
	fake.isTokenAuthorizedMutex.Unlock()
	if fake.IsTokenAuthorizedStub != nil {
		return fake.IsTokenAuthorizedStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.isTokenAuthorizedReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) IsTokenAuthorizedCallCount() int {
	fake.isTokenAuthorizedMutex.RLock()
	defer fake.isTokenAuthorizedMutex.RUnlock()
	return len(fake.isTokenAuthorizedArgsForCall)
}

func (fake *FakeCFClient) IsTokenAuthorizedCalls(stub func(string, string) (bool, error)) {
	fake.isTokenAuthorizedMutex.Lock()
	defer fake.isTokenAuthorizedMutex.Unlock()
	fake.IsTokenAuthorizedStub = stub
}

func (fake *FakeCFClient) IsTokenAuthorizedArgsForCall(i int) (string, string) {
	fake.isTokenAuthorizedMutex.RLock()
	defer fake.isTokenAuthorizedMutex.RUnlock()
	argsForCall := fake.isTokenAuthorizedArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCFClient) IsTokenAuthorizedReturns(result1 bool, result2 error) {
	fake.isTokenAuthorizedMutex.Lock()
	defer fake.isTokenAuthorizedMutex.Unlock()
	fake.IsTokenAuthorizedStub = nil
	fake.isTokenAuthorizedReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) IsTokenAuthorizedReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isTokenAuthorizedMutex.Lock()
	defer fake.isTokenAuthorizedMutex.Unlock()
	fake.IsTokenAuthorizedStub = nil
	if fake.isTokenAuthorizedReturnsOnCall == nil {
		fake.isTokenAuthorizedReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isTokenAuthorizedReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) IsUserAdmin(arg1 string) (bool, error) {
	fake.isUserAdminMutex.Lock()
	ret, specificReturn := fake.isUserAdminReturnsOnCall[len(fake.isUserAdminArgsForCall)]
	fake.isUserAdminArgsForCall = append(fake.isUserAdminArgsForCall, struct {
		arg1 string
	}{arg1})
	fake.recordInvocation("IsUserAdmin", []interface{}{arg1})
	fake.isUserAdminMutex.Unlock()
	if fake.IsUserAdminStub != nil {
		return fake.IsUserAdminStub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.isUserAdminReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) IsUserAdminCallCount() int {
	fake.isUserAdminMutex.RLock()
	defer fake.isUserAdminMutex.RUnlock()
	return len(fake.isUserAdminArgsForCall)
}

func (fake *FakeCFClient) IsUserAdminCalls(stub func(string) (bool, error)) {
	fake.isUserAdminMutex.Lock()
	defer fake.isUserAdminMutex.Unlock()
	fake.IsUserAdminStub = stub
}

func (fake *FakeCFClient) IsUserAdminArgsForCall(i int) string {
	fake.isUserAdminMutex.RLock()
	defer fake.isUserAdminMutex.RUnlock()
	argsForCall := fake.isUserAdminArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeCFClient) IsUserAdminReturns(result1 bool, result2 error) {
	fake.isUserAdminMutex.Lock()
	defer fake.isUserAdminMutex.Unlock()
	fake.IsUserAdminStub = nil
	fake.isUserAdminReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) IsUserAdminReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isUserAdminMutex.Lock()
	defer fake.isUserAdminMutex.Unlock()
	fake.IsUserAdminStub = nil
	if fake.isUserAdminReturnsOnCall == nil {
		fake.isUserAdminReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isUserAdminReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) IsUserSpaceDeveloper(arg1 string, arg2 string) (bool, error) {
	fake.isUserSpaceDeveloperMutex.Lock()
	ret, specificReturn := fake.isUserSpaceDeveloperReturnsOnCall[len(fake.isUserSpaceDeveloperArgsForCall)]
	fake.isUserSpaceDeveloperArgsForCall = append(fake.isUserSpaceDeveloperArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("IsUserSpaceDeveloper", []interface{}{arg1, arg2})
	fake.isUserSpaceDeveloperMutex.Unlock()
	if fake.IsUserSpaceDeveloperStub != nil {
		return fake.IsUserSpaceDeveloperStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.isUserSpaceDeveloperReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) IsUserSpaceDeveloperCallCount() int {
	fake.isUserSpaceDeveloperMutex.RLock()
	defer fake.isUserSpaceDeveloperMutex.RUnlock()
	return len(fake.isUserSpaceDeveloperArgsForCall)
}

func (fake *FakeCFClient) IsUserSpaceDeveloperCalls(stub func(string, string) (bool, error)) {
	fake.isUserSpaceDeveloperMutex.Lock()
	defer fake.isUserSpaceDeveloperMutex.Unlock()
	fake.IsUserSpaceDeveloperStub = stub
}

func (fake *FakeCFClient) IsUserSpaceDeveloperArgsForCall(i int) (string, string) {
	fake.isUserSpaceDeveloperMutex.RLock()
	defer fake.isUserSpaceDeveloperMutex.RUnlock()
	argsForCall := fake.isUserSpaceDeveloperArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCFClient) IsUserSpaceDeveloperReturns(result1 bool, result2 error) {
	fake.isUserSpaceDeveloperMutex.Lock()
	defer fake.isUserSpaceDeveloperMutex.Unlock()
	fake.IsUserSpaceDeveloperStub = nil
	fake.isUserSpaceDeveloperReturns = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) IsUserSpaceDeveloperReturnsOnCall(i int, result1 bool, result2 error) {
	fake.isUserSpaceDeveloperMutex.Lock()
	defer fake.isUserSpaceDeveloperMutex.Unlock()
	fake.IsUserSpaceDeveloperStub = nil
	if fake.isUserSpaceDeveloperReturnsOnCall == nil {
		fake.isUserSpaceDeveloperReturnsOnCall = make(map[int]struct {
			result1 bool
			result2 error
		})
	}
	fake.isUserSpaceDeveloperReturnsOnCall[i] = struct {
		result1 bool
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) Login() error {
	fake.loginMutex.Lock()
	ret, specificReturn := fake.loginReturnsOnCall[len(fake.loginArgsForCall)]
	fake.loginArgsForCall = append(fake.loginArgsForCall, struct {
	}{})
	fake.recordInvocation("Login", []interface{}{})
	fake.loginMutex.Unlock()
	if fake.LoginStub != nil {
		return fake.LoginStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.loginReturns
	return fakeReturns.result1
}

func (fake *FakeCFClient) LoginCallCount() int {
	fake.loginMutex.RLock()
	defer fake.loginMutex.RUnlock()
	return len(fake.loginArgsForCall)
}

func (fake *FakeCFClient) LoginCalls(stub func() error) {
	fake.loginMutex.Lock()
	defer fake.loginMutex.Unlock()
	fake.LoginStub = stub
}

func (fake *FakeCFClient) LoginReturns(result1 error) {
	fake.loginMutex.Lock()
	defer fake.loginMutex.Unlock()
	fake.LoginStub = nil
	fake.loginReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFClient) LoginReturnsOnCall(i int, result1 error) {
	fake.loginMutex.Lock()
	defer fake.loginMutex.Unlock()
	fake.LoginStub = nil
	if fake.loginReturnsOnCall == nil {
		fake.loginReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.loginReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFClient) RefreshAuthToken() (string, error) {
	fake.refreshAuthTokenMutex.Lock()
	ret, specificReturn := fake.refreshAuthTokenReturnsOnCall[len(fake.refreshAuthTokenArgsForCall)]
	fake.refreshAuthTokenArgsForCall = append(fake.refreshAuthTokenArgsForCall, struct {
	}{})
	fake.recordInvocation("RefreshAuthToken", []interface{}{})
	fake.refreshAuthTokenMutex.Unlock()
	if fake.RefreshAuthTokenStub != nil {
		return fake.RefreshAuthTokenStub()
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.refreshAuthTokenReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeCFClient) RefreshAuthTokenCallCount() int {
	fake.refreshAuthTokenMutex.RLock()
	defer fake.refreshAuthTokenMutex.RUnlock()
	return len(fake.refreshAuthTokenArgsForCall)
}

func (fake *FakeCFClient) RefreshAuthTokenCalls(stub func() (string, error)) {
	fake.refreshAuthTokenMutex.Lock()
	defer fake.refreshAuthTokenMutex.Unlock()
	fake.RefreshAuthTokenStub = stub
}

func (fake *FakeCFClient) RefreshAuthTokenReturns(result1 string, result2 error) {
	fake.refreshAuthTokenMutex.Lock()
	defer fake.refreshAuthTokenMutex.Unlock()
	fake.RefreshAuthTokenStub = nil
	fake.refreshAuthTokenReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) RefreshAuthTokenReturnsOnCall(i int, result1 string, result2 error) {
	fake.refreshAuthTokenMutex.Lock()
	defer fake.refreshAuthTokenMutex.Unlock()
	fake.RefreshAuthTokenStub = nil
	if fake.refreshAuthTokenReturnsOnCall == nil {
		fake.refreshAuthTokenReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.refreshAuthTokenReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeCFClient) SetAppInstances(arg1 string, arg2 int) error {
	fake.setAppInstancesMutex.Lock()
	ret, specificReturn := fake.setAppInstancesReturnsOnCall[len(fake.setAppInstancesArgsForCall)]
	fake.setAppInstancesArgsForCall = append(fake.setAppInstancesArgsForCall, struct {
		arg1 string
		arg2 int
	}{arg1, arg2})
	fake.recordInvocation("SetAppInstances", []interface{}{arg1, arg2})
	fake.setAppInstancesMutex.Unlock()
	if fake.SetAppInstancesStub != nil {
		return fake.SetAppInstancesStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.setAppInstancesReturns
	return fakeReturns.result1
}

func (fake *FakeCFClient) SetAppInstancesCallCount() int {
	fake.setAppInstancesMutex.RLock()
	defer fake.setAppInstancesMutex.RUnlock()
	return len(fake.setAppInstancesArgsForCall)
}

func (fake *FakeCFClient) SetAppInstancesCalls(stub func(string, int) error) {
	fake.setAppInstancesMutex.Lock()
	defer fake.setAppInstancesMutex.Unlock()
	fake.SetAppInstancesStub = stub
}

func (fake *FakeCFClient) SetAppInstancesArgsForCall(i int) (string, int) {
	fake.setAppInstancesMutex.RLock()
	defer fake.setAppInstancesMutex.RUnlock()
	argsForCall := fake.setAppInstancesArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeCFClient) SetAppInstancesReturns(result1 error) {
	fake.setAppInstancesMutex.Lock()
	defer fake.setAppInstancesMutex.Unlock()
	fake.SetAppInstancesStub = nil
	fake.setAppInstancesReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFClient) SetAppInstancesReturnsOnCall(i int, result1 error) {
	fake.setAppInstancesMutex.Lock()
	defer fake.setAppInstancesMutex.Unlock()
	fake.SetAppInstancesStub = nil
	if fake.setAppInstancesReturnsOnCall == nil {
		fake.setAppInstancesReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.setAppInstancesReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeCFClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getAppMutex.RLock()
	defer fake.getAppMutex.RUnlock()
	fake.getEndpointsMutex.RLock()
	defer fake.getEndpointsMutex.RUnlock()
	fake.getServiceInstancesInOrgMutex.RLock()
	defer fake.getServiceInstancesInOrgMutex.RUnlock()
	fake.getTokensMutex.RLock()
	defer fake.getTokensMutex.RUnlock()
	fake.isTokenAuthorizedMutex.RLock()
	defer fake.isTokenAuthorizedMutex.RUnlock()
	fake.isUserAdminMutex.RLock()
	defer fake.isUserAdminMutex.RUnlock()
	fake.isUserSpaceDeveloperMutex.RLock()
	defer fake.isUserSpaceDeveloperMutex.RUnlock()
	fake.loginMutex.RLock()
	defer fake.loginMutex.RUnlock()
	fake.refreshAuthTokenMutex.RLock()
	defer fake.refreshAuthTokenMutex.RUnlock()
	fake.setAppInstancesMutex.RLock()
	defer fake.setAppInstancesMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeCFClient) recordInvocation(key string, args []interface{}) {
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

var _ cf.CFClient = new(FakeCFClient)
