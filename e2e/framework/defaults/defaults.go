package defaults

const (
	// BandwagonOrganization specifies the name of the test organization or site to use in bandwagon form
	BandwagonOrganization = "Robotest"
	// BandwagonEmail specifies the email of the test user to use in bandwagon form
	BandwagonEmail = "robotest@example.com"
	// BandwagonUsername specifies the name of the test user to use in bandwagon form
	BandwagonUsername = "robotest"
	// BandwagonPassword specifies the password to use in bandwagon form
	BandwagonPassword = "r0b0t@st"

	// GravityHTTPPort specifies the port used by the local gravity site HTTP endpoint
	// to speed up testing (by avoiding the wait for the Load Balancer to come online)
	GravityHTTPPort = 32009
)

// ClusterAddressType defines access type to the web page for installed cluster
type ClusterAddressType string

const (
	// LoadBalancer - use loadbalancer address, works only with terraform provider
	LoadBalancer ClusterAddressType = "loadbalancer"
	// Direct - use cluster endpoints from OpsCenter cluster page
	Direct ClusterAddressType = "direct"
	// Public - use public IP addresses, works only with terraform provider
	Public ClusterAddressType = "public"
)
