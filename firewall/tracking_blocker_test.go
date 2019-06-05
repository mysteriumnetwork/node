package firewall

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionTrafficBlockIsAddedAndRemoved(t *testing.T) {
	blocker, vendor := setupBlockerAndVendor()

	removeRule, _ := blocker.BlockOutgoingTraffic(Session)
	assert.Equal(t, 1, vendor.requests["block-traffic"])
	removeRule()
	assert.Equal(t, 0, vendor.requests["block-traffic"])
}

func TestSessionTrafficBlockIsNoopWhenGlobalBlockWasCalled(t *testing.T) {
	blocker, vendor := setupBlockerAndVendor()

	removeGlobalBlock, _ := blocker.BlockOutgoingTraffic(Global)
	assert.Equal(t, 1, vendor.requests["block-traffic"])

	removeSessionRule, _ := blocker.BlockOutgoingTraffic(Session)
	assert.Equal(t, 1, vendor.requests["block-traffic"])

	removeSessionRule()
	assert.Equal(t, 1, vendor.requests["block-traffic"])

	removeGlobalBlock()
	assert.Equal(t, 0, vendor.requests["block-traffic"])
}

func TestAllowIPAccessIsAddedAndRemoved(t *testing.T) {
	blocker, vendor := setupBlockerAndVendor()

	removeRule, _ := blocker.AllowIPAccess("test-ip")
	assert.Equal(t, 1, vendor.requests["allow:test-ip"])
	removeRule()
	assert.Equal(t, 0, vendor.requests["allow:test-ip"])
}

func TestHostsFromMultipleURLsAreAllowed(t *testing.T) {
	blocker, vendor := setupBlockerAndVendor()

	removeRules, _ := blocker.AllowURLAccess("http://url1", "my-schema://url2:500/ignoredpath?ignoredQuery=true")
	assert.Equal(
		t,
		map[string]int{
			"allow:url1": 1,
			"allow:url2": 1,
		},
		vendor.requests,
	)
	removeRules()
	assert.Equal(t,
		map[string]int{
			"allow:url1": 0,
			"allow:url2": 0,
		},
		vendor.requests,
	)
}

func TestRuleIsRemovedOnlyAfterLastRemovalCall(t *testing.T) {
	blocker, vendor := setupBlockerAndVendor()

	//two independent allow requests for the same service
	removalRequest1, _ := blocker.AllowIPAccess("service")
	removalRequest2, _ := blocker.AllowIPAccess("service")
	//make sure allow ip was called once
	assert.Equal(t, 1, vendor.requests["allow:service"])
	//first removal should have no effect
	removalRequest1()
	assert.Equal(t, 1, vendor.requests["allow:service"])
	//second removal removes added rule
	removalRequest2()
	assert.Equal(t, 0, vendor.requests["allow:service"])
}

func setupBlockerAndVendor() (*referenceTrackingBlocker, *mockedVendor) {
	vendor := &mockedVendor{
		requests: make(map[string]int),
	}
	blocker := newTrackingBlocker()
	blocker.SwitchVendor(vendor)
	return blocker, vendor
}

type mockedVendor struct {
	requests map[string]int
}

func (mv *mockedVendor) BlockOutgoingTraffic() (RemoveRule, error) {
	return mv.increaseRef("block-traffic")
}

func (mv *mockedVendor) AllowIPAccess(ip string) (RemoveRule, error) {
	return mv.increaseRef("allow:" + ip)
}

func (mockedVendor) Reset() {

}

func (mv *mockedVendor) increaseRef(ref string) (RemoveRule, error) {
	mv.requests[ref] = mv.requests[ref] + 1
	return mv.decreaseRef(ref), nil
}

func (mv *mockedVendor) decreaseRef(ref string) RemoveRule {
	return func() {
		mv.requests[ref] = mv.requests[ref] - 1
	}
}

var _ Vendor = (*mockedVendor)(nil)
