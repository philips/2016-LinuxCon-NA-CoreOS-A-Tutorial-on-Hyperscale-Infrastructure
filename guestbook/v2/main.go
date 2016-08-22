/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/xyproto/simpleredis"
)

var (
	masterPool *simpleredis.ConnectionPool
	slavePool  *simpleredis.ConnectionPool
)

func ListRangeHandler(rw http.ResponseWriter, req *http.Request) {
	key := mux.Vars(req)["key"]
	list := simpleredis.NewList(slavePool, key)
	members := HandleError(list.GetAll()).([]string)
	membersJSON := HandleError(json.MarshalIndent(members, "", "  ")).([]byte)
	rw.Write(membersJSON)
}

func ListPushHandler(rw http.ResponseWriter, req *http.Request) {
	key := mux.Vars(req)["key"]
	value := mux.Vars(req)["value"]
	list := simpleredis.NewList(masterPool, key)
	HandleError(nil, list.Add(value))
	ListRangeHandler(rw, req)
}

func InfoHandler(rw http.ResponseWriter, req *http.Request) {
	info := HandleError(masterPool.Get(0).Do("INFO")).([]byte)
	rw.Write(info)
}

func EnvHandler(rw http.ResponseWriter, req *http.Request) {
	environment := make(map[string]string)
	for _, item := range os.Environ() {
		splits := strings.Split(item, "=")
		key := splits[0]
		val := strings.Join(splits[1:], "=")
		environment[key] = val
	}

	envJSON := HandleError(json.MarshalIndent(environment, "", "  ")).([]byte)
	rw.Write(envJSON)
}

func HandleError(result interface{}, err error) (r interface{}) {
	if err != nil {
		panic(err)
	}
	return result
}

func HandleTwilio() {
	c := time.Tick(100 * time.Millisecond)
	for range c {
		findMessages()
	}
}

type sentMessages struct {
	Number string
	Last   int
}

func findMessages() {
	outbox := simpleredis.NewKeyValue(masterPool, "outbox")
	phoneNumbers, err := simpleredis.NewList(slavePool, "phoneNumbers").GetAll()
	if err != nil {
		fmt.Println(err)
	}
	entries, err := simpleredis.NewList(slavePool, "guestbook").GetAll()
	if err != nil {
		fmt.Println(err)
	}

	for _, n := range phoneNumbers {
		last, err := outbox.Get(n)
		if err != nil {
			fmt.Println(err)
			continue
		}
		l, err := strconv.Atoi(last)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if len(entries) < l {
			continue
		}

		last, err = outbox.Inc(n)
		l, err = strconv.Atoi(last)
		if err != nil {
			fmt.Println(err)
			continue
		}

		for _, e := range entries[(l - 2):(l - 1)] {
			sendTwilio(n, e)
		}
	}
}

func sendTwilio(number string, msg string) {
	// Set initial variables
	accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
	authToken := os.Getenv("TWILIO_ACCOUNT_TOKEN")
	if authToken == "" || accountSid == "" {
		fmt.Printf("empty accountSid or authToken, not using Twilio number=%v msg=%v\n", number, msg)
		return
	}
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + accountSid + "/Messages.json"

	// Build out the data for our message
	v := url.Values{}
	v.Set("To", number)
	v.Set("From", "+14157874263")

	end := len(msg)
	if end > 110 {
		end = 110
	}
	m := msg[:end]
	v.Set("Body", m+" To stop reply STOP")
	rb := *strings.NewReader(v.Encode())

	// Create client
	client := &http.Client{}

	req, _ := http.NewRequest("POST", urlStr, &rb)
	req.SetBasicAuth(accountSid, authToken)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make request
	resp, _ := client.Do(req)
	fmt.Println(resp.Status)
	//fmt.Printf("%v\n", req)
}

func main() {
	master := os.Getenv("REDIS_MASTER")
	if master == "" {
		master = "redis-master:6379"
	}
	masterPool = simpleredis.NewConnectionPoolHost(master)
	defer masterPool.Close()

	slave := os.Getenv("REDIS_SLAVE")
	if slave == "" {
		slave = "redis-slave:6379"
	}
	slavePool = simpleredis.NewConnectionPoolHost(slave)
	defer slavePool.Close()

	r := mux.NewRouter()
	r.Path("/lrange/{key}").Methods("GET").HandlerFunc(ListRangeHandler)
	r.Path("/rpush/{key}/{value}").Methods("GET").HandlerFunc(ListPushHandler)
	r.Path("/info").Methods("GET").HandlerFunc(InfoHandler)
	r.Path("/env").Methods("GET").HandlerFunc(EnvHandler)

	go HandleTwilio()

	n := negroni.Classic()
	n.UseHandler(r)
	n.Run(":3000")
}
