package types

// VerifiedIdentity contains only the trusted claims needed after token
// verification.
type VerifiedIdentity struct {
	Issuer  string
	Subject string
	Roles   []string
}

// Capability is an application permission derived from trusted identity roles.
type Capability uint32

const (
	CapabilityInventoryRead Capability = 1 << iota
	CapabilityInventoryManage
	CapabilityUsersManage
	CapabilityCheckoutSelf
	CapabilityCheckoutManage
	CapabilityCheckoutHistoryReadAll
)

// CapabilitySet is an immutable value containing application permissions.
type CapabilitySet uint32

// NewCapabilitySet builds a capability set without retaining mutable state.
func NewCapabilitySet(capabilities ...Capability) CapabilitySet {
	var set CapabilitySet
	for _, capability := range capabilities {
		set |= CapabilitySet(capability)
	}
	return set
}

// Has reports whether the set includes the requested capability.
func (s CapabilitySet) Has(capability Capability) bool {
	return s&CapabilitySet(capability) != 0
}

// Union returns a new set containing permissions from both sets.
func (s CapabilitySet) Union(other CapabilitySet) CapabilitySet {
	return s | other
}
