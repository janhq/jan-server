package registry

var SchemaRegistry []interface{}

func RegisterSchemaForAutoMigrate(models ...interface{}) {
	SchemaRegistry = append(SchemaRegistry, models...)
}
