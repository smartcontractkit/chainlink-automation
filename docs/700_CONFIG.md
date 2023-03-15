# Config

The config should be an abstraction for getting specific config values from
different versions of an off-chain config.

```
type IConfig interface {
    // Has simply indicates that the config has a specific value or not
    Has(string) bool
    // Get provides the means to get any value of any type from a configuration
    // and an error can indicate type issues
    Get(string, interface{}) error
}
```

The constructor of a Config should take an interface of type struct or 
`map[string]interface{}`. This will allow a variety of off-chain configurations
to be created and passed from layer to layer.