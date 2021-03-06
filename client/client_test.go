// +build ignore

package client

import (
	"github.com/golang/glog"
	"github.com/heidi-ann/ios/config"
	"github.com/heidi-ann/ios/ios/server"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestStartClient(t *testing.T) {
	assert := assert.New(t)
	//Create temp directories
	dirClient, err := ioutil.TempDir("", "IosStartClientTests")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirClient)

	//start 3 node Ios cluster
	serverConfigFile := os.Getenv("GOPATH") + "/src/github.com/heidi-ann/ios/ios/example3.conf"
	for id := 0; id <= 2; id++ {
		dirServer, err := ioutil.TempDir("", "IosStartClientTests")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(dirServer)
		go server.RunIos(id, config.ParseServerConfig(serverConfigFile), dirServer)
	}

	//start client
	clientConfigFile := os.Getenv("GOPATH") + "/src/github.com/heidi-ann/ios/client/example3.conf"

	client := StartClientFromConfigFile(1, dirClient+"/latency.csv", clientConfigFile)

	//submit requests
	success, reply, err := client.SubmitRequest("update A 1")
	assert.Nil(err)
	assert.True(success, "Request not successful")
	assert.Equal("OK", reply, "Response not as expected")

	success, reply, err = client.SubmitRequest("get A")
	assert.Nil(err)
	assert.True(success, "Request not successful")
	assert.Equal("1", reply, "Response not as expected")

	client2 := StartClientFromConfigFile(2, dirClient+"/latency2.csv", clientConfigFile)

	//submit requests to new client
	success, reply, err = client2.SubmitRequest("get A")
	assert.Nil(err)
	assert.True(success, "Request not successful")
	assert.Equal("1", reply, "Response not as expected")

	success, reply, err = client2.SubmitRequest("update B 2")
	assert.Nil(err)
	assert.True(success, "Request not successful")
	assert.Equal("OK", reply, "Response not as expected")

	//check original client is still ok
	success, reply, err = client.SubmitRequest("get B")
	assert.Nil(err)
	assert.True(success, "Request not successful")
	assert.Equal("2", reply, "Response not as expected")

	//stopping client
	client.StopClient()
	client2.StopClient()

}
