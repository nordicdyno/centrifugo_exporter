package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/centrifugal/centrifugo/libcentrifugo/auth"
	"github.com/centrifugal/gocent"
)

type NodeResponse struct {
	Data NodeData
}

type NodeData struct {
	Metrics NodeMetrics
}

type NodeMetrics map[string]float64

func nodeMetrics(c *gocent.Client) (NodeMetrics, error) {
	cmds := []gocent.Command{
		{Method: "node"},
	}
	raw, err := send(c, cmds)
	if err != nil {
		return nil, err
	}
	resp := raw[0]
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	// fmt.Printf("RESULT => %v\n", string(resp.Body))

	return decodeNodeStat(resp.Body)
}

func decodeNodeStat(body []byte) (NodeMetrics, error) {
	var respJSON NodeResponse
	err := json.Unmarshal(body, &respJSON)
	if err != nil {
		return nil, err
	}
	return respJSON.Data.Metrics, nil
}

func send(c *gocent.Client, cmds []gocent.Command) (gocent.Result, error) {
	data, err := json.Marshal(cmds)
	if err != nil {
		return gocent.Result{}, err
	}

	client := &http.Client{}
	client.Timeout = c.Timeout
	r, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(data))
	if err != nil {
		return gocent.Result{}, err
	}

	r.Header.Set("X-API-Sign", auth.GenerateApiSign(c.Secret, data))
	r.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(r)
	if err != nil {
		return gocent.Result{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return gocent.Result{}, errors.New("wrong status code: " + resp.Status)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var result gocent.Result
	err = json.Unmarshal(body, &result)
	return result, err
}
