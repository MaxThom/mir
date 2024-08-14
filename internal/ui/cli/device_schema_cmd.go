package cli

type DeviceSchemaCmd struct {
	Upload  SchemaUploadCmd  `cmd:"" help:"Upload schema of a set of devices"`
	Explore SchemaExploreCmd `cmd:"" help:"Explore a device schema"`
}

type SchemaUploadCmd struct {
	Target `embed:"" prefix:"target."`
	// TODO could be an array, I think there is a path type
	Path string `help:"Path to protobuf schema"`
}

type SchemaExploreCmd struct {
	// TODO could be target, and we download multiple files
	TargetDeviceId string `short:"d" help:"DeviceID to retrieve schema from"`
}

func (d *SchemaUploadCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if len(d.Target.Ids) == 0 &&
		len(d.Target.Names) == 0 &&
		len(d.Target.Namespaces) == 0 &&
		len(d.Target.Labels) == 0 &&
		len(d.Target.Anno) == 0 {
		err.Details = append(err.Details, "Must specify targets")
	}

	if d.Path == "" {
		err.Details = append(err.Details, "Invalid protobuf schema path")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *SchemaUploadCmd) Run(c CLI) error {

	return nil
}

func (d *SchemaExploreCmd) Validate() error {
	err := MirInvalidInputError{
		Details: []string{},
	}

	if d.TargetDeviceId == "" {
		err.Details = append(err.Details, "Must specify a device id")
	}

	if len(err.Details) > 0 {
		return err
	}
	return nil
}

func (d *SchemaExploreCmd) Run(c CLI) error {

	return nil
}
