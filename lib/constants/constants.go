package constants

const (
	// FieldCommandErrorReport defines a logging field to store the error message for a failed command
	FieldCommandErrorReport = "errmsg"

	// FieldCommandError defines a logging field that determines if the command has failed
	FieldCommandError = "cmderr"

	// FieldProvisioner defines a logging field to specify the name of the used provisioner
	FieldProvisioner = "provisioner"

	// FieldCluster defines a logging field to specify the name of the cluster
	FieldCluster = "cluster"

	// SharedDirMask is a mask for shared directories
	SharedDirMask = 0755

	// SharedReadMask is a mask for shared directories
	SharedReadMask = 0644

	// SharedReadWriteMask is a mask for a shared file with world read/write access
	SharedReadWriteMask = 0666

	// EnvDockerDevice specifies the name of the environment variable with docker device name
	EnvDockerDevice = "DOCKER_DEVICE"

	// OSRedHat is RedHat Enterprise Linux
	OSRedHat = "redhat"

	// DeviceMapper is devicemapper storage driver name
	DeviceMapper = "devicemapper"

	// Overlay is overlay storage driver name
	Overlay = "overlay"

	// Overlay2 is version 2 of overlay storage driver
	Overlay2 = "overlay2"

	// Loopback is local storage
	Loopback = "loopback"

	// ManifestStorageDriver is empty string identifying that install should use driver defined by the manifest
	ManifestStorageDriver = ""

	// Ops specifies a special cloud provider - a telekube Ops Center
	Ops = "ops"

	// TerraformPluginDir specifies the location of terraform plugins
	TerraformPluginDir = "/etc/terraform/plugins"
)

const (
	// AWS is the Amazon cloud
	AWS = "aws"
	// Azure is Microsoft Zzure cloud
	Azure = "azure"
	// GCE is Google Compute Engine cloud
	GCE = "gce"
)
