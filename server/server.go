package server

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/go-redis/redis"
	base62 "github.com/pilu/go-base62"
)

var redis_client *redis.Client

func Init() {
	fmt.Print("[server] init redis...")
	redis_client = redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})
	fmt.Print("OK\n")

	fmt.Print("[server] init http...")
	http.HandleFunc("/", index)
	http.HandleFunc("/short", api_short)

	fmt.Print("OK\n")
	fmt.Println("[server] start listening...")
	err := http.ListenAndServe("0.0.0.0:9999", nil)
	if err != nil {
		log.Fatal("http server start failed!", err)
	}
}

func index(resp http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		resp.WriteHeader(404)
		return
	}

	path := req.URL.Path
	if path == "/" {
		resp.WriteHeader(404)
		return
	}

	path = path[1:]
	ret := redis_client.HGet("url", string(base62.Decode(path)-12345))
	if ret.Err() != nil {
		resp.WriteHeader(404)
	} else {
		origin_url := ret.Val()
		log.Printf("[server] url, %s => %s", path, origin_url)
		http.Redirect(resp, req, origin_url, 301)
		// fmt.Fprintln(resp, ret.String())
	}
}

func api_short(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "PUT" {
		log.Println("[server] handle api \"short\",", req.Method)
		result, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.WriteHeader(500)
		} else {
			origin_url := strings.TrimSpace(string(result))
			if len(origin_url) <= 0 {
				log.Fatal("[server] bad url data", origin_url)
				resp.WriteHeader(500)
			} else {
				ret := redis_client.Incr("short_url_id")
				if ret.Err() != nil {
					log.Fatal("[server] failed to prepare record", ret.Err())
					resp.WriteHeader(500)
				} else {
					url_id := ret.Val()
					ret := redis_client.HSet("url", string(url_id), origin_url)
					if ret.Err() != nil {
						log.Fatal("[server] failed to add record", ret.Err())
						resp.WriteHeader(500)
					} else {
						// 把id编码构造url下发
						// FIXME: 潜在的溢出，暂时不用管，64位系统上应该没有关系
						short_id := base62.Encode(int(url_id + 12345))
						fmt.Fprintf(resp, "https://lyly.ws/%s", short_id)
					}
				}
			}
		}
	} else {
		resp.WriteHeader(404)
	}
}
