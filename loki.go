package main

import "github.com/cortexproject/cortex/pkg/util/flagext"
import "github.com/prometheus/common/model"

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type lokiConfig struct {
	url           flagext.URLValue
	batchWait     time.Duration
	batchSize     int
	extraLabels   model.LabelSet
	labelKeys     []string
	dropSingleKey bool
}

type labelSetJSON struct {
	Labels []struct {
		Key   string `json:"key"`
		Label string `json:"label"`
	} `json:"labels"`
}

func getLokiConfig(url string, batchWait string, batchSize string, extraLabels string, labelKeys string, dropSingleKey string) (*lokiConfig, error) {
	lc := &lokiConfig{}
	var clientURL flagext.URLValue
	if url == "" {
		url = "http://localhost:3100/api/prom/push"
	}
	err := clientURL.Set(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse client URL")
	}
	lc.url = clientURL

	batchWaitValue, err := strconv.Atoi(batchWait)
	if err != nil || batchWait == "" {
		batchWaitValue = 10
	}
	lc.batchWait = time.Duration(batchWaitValue) * time.Millisecond

	batchSizeValue, err := strconv.Atoi(batchSize)
	if err != nil || batchSize == "" {
		batchSizeValue = 10
	}
	lc.batchSize = batchSizeValue * 1024

	var labelValues labelSetJSON
	if extraLabels == "" {
		extraLabels = `{"labels": [{"key": "job", "label": "fluent-bit"}]}`
	}

	json.Unmarshal(([]byte)(extraLabels), &labelValues)
	lc.extraLabels = make(model.LabelSet)
	for _, v := range labelValues.Labels {
		lc.extraLabels[model.LabelName(v.Key)] = model.LabelValue(v.Label)
	}

	lc.labelKeys = strings.Split(labelKeys, ",")
	for i, v := range lc.labelKeys {
		lc.labelKeys[i] = strings.Trim(v, " ")
	}

	lc.dropSingleKey = (dropSingleKey == "true") || (dropSingleKey == "1")

	return lc, nil
}
