package server

import (
	"encoding/json"
	"github.com/urfave/cli"
	"github.com/vmware/transport-go/bus"
	"github.com/vmware/transport-go/plank/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

// generatePlatformServerConfig is a generic internal method that returns the pointer of a new
// instance of PlatformServerConfig. for an argument it can be passed either *serverConfigFactory
// or *cli.Context which the method will analyze and determine the best way to extract user provided values from it.
func generatePlatformServerConfig(i interface{}) (*PlatformServerConfig, error) {
	configFile := extractFlagValueFromProvider(i, "ConfigFile", "string").(string)
	host := extractFlagValueFromProvider(i, "Hostname", "string").(string)
	port := extractFlagValueFromProvider(i, "Port", "int").(int)
	rootDir := extractFlagValueFromProvider(i, "RootDir", "string").(string)
	static := extractFlagValueFromProvider(i, "Static", "[]string").([]string)
	shutdownTimeoutInMinutes := extractFlagValueFromProvider(i, "ShutdownTimeout", "int64").(int64)
	accessLog := extractFlagValueFromProvider(i, "AccessLog", "string").(string)
	outputLog := extractFlagValueFromProvider(i, "OutputLog", "string").(string)
	errorLog := extractFlagValueFromProvider(i, "ErrorLog", "string").(string)
	debug := extractFlagValueFromProvider(i, "Debug", "bool").(bool)
	noBanner := extractFlagValueFromProvider(i, "NoBanner", "bool").(bool)
	cert := extractFlagValueFromProvider(i, "Cert", "string").(string)
	certKey := extractFlagValueFromProvider(i, "CertKey", "string").(string)
	spaPath := extractFlagValueFromProvider(i, "SpaPath", "string").(string)
	noFabricBroker := extractFlagValueFromProvider(i, "NoFabricBroker", "bool").(bool)
	fabricEndpoint := extractFlagValueFromProvider(i, "FabricEndpoint", "string").(string)
	topicPrefix := extractFlagValueFromProvider(i, "TopicPrefix", "string").(string)
	queuePrefix := extractFlagValueFromProvider(i, "QueuePrefix", "string").(string)
	requestPrefix := extractFlagValueFromProvider(i, "RequestPrefix", "string").(string)
	requestQueuePrefix := extractFlagValueFromProvider(i, "RequestQueuePrefix", "string").(string)
	prometheus := extractFlagValueFromProvider(i, "Prometheus", "bool").(bool)
	restBridgeTimeout := extractFlagValueFromProvider(i, "RestBridgeTimeout", "int64").(int64)

	// if config file flag is provided, read directly from the file
	if len(configFile) > 0 {
		var serverConfig PlatformServerConfig
		b, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(b, &serverConfig); err != nil {
			return nil, err
		}

		// handle invalid duration by setting it to the default value of 1 minute
		if serverConfig.RestBridgeTimeoutInMinutes <= 0 {
			serverConfig.RestBridgeTimeoutInMinutes = 1
		}

		// the raw value from the config.json needs to be multiplied by time.Minute otherwise it's interpreted as nanosecond
		serverConfig.RestBridgeTimeoutInMinutes = serverConfig.RestBridgeTimeoutInMinutes * time.Minute

		return &serverConfig, nil
	}

	// handle invalid duration by setting it to the default value of 1 minute
	if restBridgeTimeout <= 0 {
		restBridgeTimeout = 1
	}

	// instantiate a server config
	serverConfig := &PlatformServerConfig{
		Host:                     host,
		Port:                     port,
		RootDir:                  rootDir,
		StaticDir:                static,
		ShutdownTimeoutInMinutes: time.Duration(shutdownTimeoutInMinutes),
		LogConfig: &utils.LogConfig{
			AccessLog:     accessLog,
			ErrorLog:      errorLog,
			OutputLog:     outputLog,
			Root:          rootDir,
			FormatOptions: &utils.LogFormatOption{},
		},
		Debug:                      debug,
		NoBanner:                   noBanner,
		EnablePrometheus:           prometheus,
		RestBridgeTimeoutInMinutes: time.Duration(restBridgeTimeout) * time.Minute,
	}

	if len(certKey) > 0 && len(certKey) > 0 {
		var err error
		certKey, err = filepath.Abs(certKey)
		if err != nil {
			return nil, err
		}
		cert, err = filepath.Abs(cert)
		if err != nil {
			return nil, err
		}

		serverConfig.TLSCertConfig = &TLSCertConfig{CertFile: cert, KeyFile: certKey}
	}

	if len(strings.TrimSpace(spaPath)) > 0 {
		var err error
		serverConfig.SpaConfig, err = NewSpaConfig(spaPath)
		if err != nil {
			return nil, err
		}
	}

	// unless --no-fabric-broker flag is provided, set up a broker config
	if !noFabricBroker {
		serverConfig.FabricConfig = &FabricBrokerConfig{
			FabricEndpoint: fabricEndpoint,
			EndpointConfig: &bus.EndpointConfig{
				TopicPrefix:           topicPrefix,
				UserQueuePrefix:       queuePrefix,
				AppRequestPrefix:      requestPrefix,
				AppRequestQueuePrefix: requestQueuePrefix,
				Heartbeat:             60000},
		}
	}

	return serverConfig, nil
}

// marshalResponseBody takes body as an interface not knowing whether it is already converted to []byte or not.
// if it is of a map type then it marshals it using json.Marshal to get the byte representation of it. otherwise
// the input is cast to []byte and returned.
func marshalResponseBody(body interface{}) (bytes []byte, err error) {
	vt := reflect.TypeOf(body)
	if vt == reflect.TypeOf([]byte{}) {
		bytes, err = body.([]byte), nil
	} else {
		bytes, err = json.Marshal(body)
	}

	return
}

// sanitizeConfigRootPath takes *PlatformServerConfig, ensures the path specified by RootDir field exists.
// if RootDir is empty then the current working directory will be populated. if for some reason the path
// cannot be accessed it'll cause a panic.
func sanitizeConfigRootPath(config *PlatformServerConfig) {
	if len(config.RootDir) == 0 {
		wd, _ := os.Getwd()
		config.RootDir = wd
	}

	absRootPath, err := filepath.Abs(config.RootDir)
	if err != nil {
		panic(err)
	}

	_, err = os.Stat(absRootPath)
	if err != nil {
		panic(err)
	}

	// once it has been confirmed that the path exists, set config.RootDir to the absolute path
	config.RootDir = absRootPath
}

// extractFlagValueFromProvider extracts from provider a value for key. when the provider is of *cli.Context
// type parseType will be used to invoke the correct method to convert the user input to its appropriate value type.
func extractFlagValueFromProvider(provider interface{}, key string, parseType string) interface{} {
	switch provider.(type) {
	case *cli.Context:
		cast := provider.(*cli.Context)
		switch parseType {
		case "string":
			return cast.String(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "int":
			return cast.Int(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "int64":
			return cast.Int64(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "[]string":
			return cast.StringSlice(utils.PlatformServerFlagConstants[key]["FlagName"])
		case "bool":
			return cast.Bool(utils.PlatformServerFlagConstants[key]["FlagName"])
		}
		break
	case *serverConfigFactory:
		refl := reflect.ValueOf(provider)
		method := refl.MethodByName(key)
		raw := method.Call([]reflect.Value{})
		return raw[0].Interface()
	}
	return nil
}