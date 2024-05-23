### Integration test

each tests in a single package must be independent. So
independent at the package level.

Moreover, all the required infrastructure must be setup prior to running the tests. But all Mir Services must be started at the setup or at the unit test level.

example, the mir_integration_test.go file in pkgs/mir_device.
for those tests, we need NatsIO, SurrealDB and the Core service.
NatsIO and SurrealDB get started with docker prior as a requirement. The Core service and other services if needed is started in the test or the setup.
