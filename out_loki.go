package main

import "github.com/fluent/fluent-bit-go/output"
import "github.com/grafana/loki/pkg/promtail/client"
import "github.com/sirupsen/logrus"
import kit "github.com/go-kit/kit/log/logrus"
import "github.com/cortexproject/cortex/pkg/util/flagext"
import "github.com/json-iterator/go"
import "github.com/prometheus/common/model"

import (
	"C"
	"fmt"
	"os"
	"time"
	"unsafe"
)

var loki *client.Client
var config *lokiConfig
var plugin GoOutputPlugin = &fluentPlugin{}

type GoOutputPlugin interface {
	PluginConfigKey(ctx unsafe.Pointer, key string) string
	Unregister(ctx unsafe.Pointer)
	GetRecord(dec *output.FLBDecoder) (ret int, ts interface{}, rec map[interface{}]interface{})
	NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder
	HandleLine(timestamp time.Time, record map[interface{}]interface{}) error
	Exit(code int)
}

type fluentPlugin struct{}

func (p *fluentPlugin) PluginConfigKey(ctx unsafe.Pointer, key string) string {
	return output.FLBPluginConfigKey(ctx, key)
}

func (p *fluentPlugin) Unregister(ctx unsafe.Pointer) {
	output.FLBPluginUnregister(ctx)
}

func (p *fluentPlugin) GetRecord(dec *output.FLBDecoder) (int, interface{}, map[interface{}]interface{}) {
	return output.GetRecord(dec)
}

func (p *fluentPlugin) NewDecoder(data unsafe.Pointer, length int) *output.FLBDecoder {
	return output.NewDecoder(data, int(length))
}

func (p *fluentPlugin) Exit(code int) {
	os.Exit(code)
}

func (p *fluentPlugin) HandleLine(timestamp time.Time, record map[interface{}]interface{}) error {
	var err error
	var line []byte
	m := make(map[string]interface{})
	labels := config.extraLabels.Clone()

	var strValue string

RecordLoop:
	for k, v := range record {
		switch t := v.(type) {
		case []byte:
			// prevent encoding to base64
			strValue = string(t)
		default:
			strValue = v.(string)
		}

		for _, label := range config.labelKeys {
			if k == label {
				labels[model.LabelName(k.(string))] = model.LabelValue(strValue)
				continue RecordLoop
			}
		}
		m[k.(string)] = strValue
		line = []byte(strValue)
	}

	if !config.dropSingleKey {
		line, err = jsoniter.Marshal(m)
		if err != nil {
			return err
		}
	}

	return loki.Handle(labels, timestamp, string(line))
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "loki", "Loki Output plugin written in GO!")
}

//export FLBPluginInit
// (fluentbit will call this)
// ctx (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(ctx unsafe.Pointer) int {
	var err error
	config, err = getLokiConfig(
		plugin.PluginConfigKey(ctx, "URL"),
		plugin.PluginConfigKey(ctx, "BatchWait"),
		plugin.PluginConfigKey(ctx, "BatchSize"),
		plugin.PluginConfigKey(ctx, "ExtraLabels"),
		plugin.PluginConfigKey(ctx, "LabelKeys"),
		plugin.PluginConfigKey(ctx, "DropSingleKey"),
	)
	if err != nil {
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}
	fmt.Printf("[loki] URL parameter = '%s'\n", config.url)
	fmt.Printf("[loki] BatchWait parameter = '%d'\n", config.batchSize)
	fmt.Printf("[loki] BatchSize parameter = '%s'\n", config.batchWait.String())
	fmt.Printf("[loki] ExtraLabels parameter = '%s'\n", config.extraLabels)
	fmt.Printf("[loki] LabelKeys parameter = '%s'\n", config.labelKeys)
	fmt.Printf("[loki] DropSingleKey parameter = '%t'\n", config.dropSingleKey)

	cfg := client.Config{}
	// Init everything with default values.
	flagext.RegisterFlags(&cfg)

	// Override some of those defaults
	cfg.URL = config.url
	cfg.BatchWait = config.batchWait
	cfg.BatchSize = config.batchSize

	log := logrus.New()

	loki, err = client.New(cfg, kit.NewLogrusLogger(log))
	if err != nil {
		log.Fatalf("client.New: %s\n", err)
		plugin.Unregister(ctx)
		plugin.Exit(1)
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	dec := plugin.NewDecoder(data, int(length))

	for {
		ret, ts, record = plugin.GetRecord(dec)
		if ret != 0 {
			break
		}

		// Get timestamp
		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			fmt.Print("timestamp isn't known format. Use current time.")
			timestamp = time.Now()
		}


		err := plugin.HandleLine(timestamp, record)
		if err != nil {
			fmt.Printf("error sending message for Grafana Loki: %v", err)
			return output.FLB_RETRY
		}
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	loki.Stop()
	return output.FLB_OK
}

func main() {
}
