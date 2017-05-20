package defaults

import "time"

const (
	// AjaxCallTimeout specifies the amount of time needed to complete AJAX request
	AjaxCallTimeout = 20 * time.Second
	ElementTimeout  = 20 * time.Second
	// FindTimeout defines the timeout to use for lookup operations
	FindTimeout = 1 * time.Minute
	// EventuallyPollInterval defines the frequency of Eventually polling attempts
	EventuallyPollInterval = 300 * time.Millisecond

	// ProvisionerSelectedTimeout specifies the amount of time to wait for the requirements screen after selecting a provisioner
	ProvisionerSelectedTimeout = 5 * time.Minute
	// AgentServerTimeout defines the amount of time to wait for agents to connect
	AgentServerTimeout = 5 * time.Minute

	// InstallStartTimeout specifies the amount of time to wait for the install to start
	InstallStartTimeout = 5 * time.Minute
	// InstallTimeout defines the amount of time to wait for installation to complete
	InstallTimeout = 40 * time.Minute
	// InstallSuccessMessagePollInterval defines the frequency of polling attempts to retrieve installation state
	InstallSuccessMessagePollInterval = time.Second

	// SiteServerListRefreshTimeout defines amount of time needed to refresh server list table on Site Server page
	SiteServerListRefreshTimeout = 5 * time.Second
	// SiteServerListRefreshAfterShrinkTimeout defines amount of time needed to unregister server from all places
	SiteServerListRefreshAfterShrinkTimeout = 2 * time.Minute
	// SiteLogoutAfterUpdateTimeout defines amount of time needed to redirect a user to login page after update operation
	SiteLogoutAfterUpdateTimeout = 5 * time.Minute
	// SiteLogoutAfterUpdatePollInterval defines the frequency of polling attempts to check if a user has been logged out after update operation
	SiteLogoutAfterUpdatePollInterval = 5 * time.Second
	// SiteOperationTimeout time for operation to be completed (Expand and Application Update operations)
	SiteOperationTimeout = 10 * time.Minute

	// SlowOperationTimeout defines how long to wait on slow operations
	SlowOperationTimeout = 30 * time.Minute

	// OpsCenterDeleteSiteTimeout specifies the amount of time allotted to a site delete operation
	OpsCenterDeleteSiteTimeout = 5 * time.Minute

	// BandwagonOrganization specifies the name of the test organization or site to use in bandwagon form
	BandwagonOrganization = "Robotest"
	// BandwagonEmail specifies the email of the test user to use in bandwagon form
	BandwagonEmail = "robotest@example.com"
	// BandwagonUsername specifies the name of the test user to use in bandwagon form
	BandwagonUsername = "robotest"
	// BandwagonPassword specifies the password to use in bandwagon form
	BandwagonPassword = "r0b0t@st"
	// BandwagonSubmitFormTimeout defines timeout for submit form request
	BandwagonSubmitFormTimeout = 10 * time.Second

	// GravityHTTPPort specifies the port used by the local gravity site HTTP endpoint
	// to speed up testing (by avoiding the wait for the Load Balancer to come online)
	GravityHTTPPort = 32009
)
