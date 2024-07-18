# Questioning

1.
  - API Questioning with naming conflict, maybe core_api?
    Maybe a new folder under services/core/client with the
    client file with package name 'core_client', Models could also be stored there. Maybe put the clients under api/clients/v1alpha/<service>
  - Models of proto also have the annoying sync.lock,
    should I make my own objects? Feels annoying, maybe
    it's time I look at a proto plugin that can generate
    them with it? and it generate them in the client folder
    describe above, or maybe have a type folder.
  - Might also fix the test_utils conflict with the core   package because test_utils also utilize core so cycle import
2.
