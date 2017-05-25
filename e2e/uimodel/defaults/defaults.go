package defaults

import "time"

const (
	// AjaxCallTimeout specifies the amount of time needed to complete AJAX request
	AjaxCallTimeout = 20 * time.Second
	// AppLoadTimeout specifies the amount of time needed for web app to load
	AppLoadTimeout = 20 * time.Second
	// FindTimeout defines the timeout to use for lookup operations
	FindTimeout = 1 * time.Minute
	// EventuallyPollInterval defines the frequency of Eventually polling attempts
	EventuallyPollInterval = 300 * time.Millisecond

	// AgentServerTimeout defines the amount of time to wait for agents to connect
	AgentServerTimeout = 5 * time.Minute

	// InstallCreateClusterTimeout specifies the amount of time needed to create a cluster and proceed to the next step
	InstallCreateClusterTimeout = 5 * time.Minute
	// InstallStartTimeout specifies the amount of time to wait for the install to start
	InstallStartTimeout = 5 * time.Minute
	// InstallTimeout defines the amount of time to wait for installation to complete
	InstallTimeout = 40 * time.Minute
	// InstallCompletionPollInterval defines poll interval for checking install completion status
	InstallCompletionPollInterval = 5 * time.Second

	// SiteServerListRefreshTimeout defines amount of time needed to refresh server list table on Site Server page
	SiteServerListRefreshTimeout = 5 * time.Second
	// SiteServerListRefreshAfterShrinkTimeout defines amount of time needed to unregister server from all places
	SiteServerListRefreshAfterShrinkTimeout = 2 * time.Minute
	// SiteLogoutAfterUpdateTimeout defines amount of time needed to redirect a user to login page after update operation
	SiteLogoutAfterUpdateTimeout = 30 * time.Minute
	// SiteLogoutAfterUpdatePollInterval defines the frequency of polling attempts to check if a user has been logged out after update operation
	SiteLogoutAfterUpdatePollInterval = 7 * time.Second
	// SiteOperationEndTimeout time for operation to be completed (Expand and Application Update operations)
	SiteOperationEndTimeout = 10 * time.Minute
	// SiteOperationStartTimeout is a waiting time for operation to start
	SiteOperationStartTimeout = 20 * time.Second
	// SiteFetchServerProfileTimeout is a waiting time to fetch AWS server profiles
	SiteFetchServerProfileTimeout = 20 * time.Second

	// LoginGoogleNextStepTimeout specifies the amount of time needed for google auth steps to initialize
	LoginGoogleNextStepTimeout = 10 * time.Second

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
