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
)
