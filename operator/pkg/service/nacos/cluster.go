package nacosClient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type INacosClient interface {
}

type NacosClient struct {
	logger     log.Logger
	httpClient http.Client
}

type ServersInfo struct {
	Servers []struct {
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		State      string `json:"state"`
		ExtendInfo struct {
			LastRefreshTime int64 `json:"lastRefreshTime"`
			RaftMetaData    struct {
				MetaDataMap struct {
					NamingPersistentService struct {
						Leader          string   `json:"leader"`
						RaftGroupMember []string `json:"raftGroupMember"`
						Term            int      `json:"term"`
					} `json:"naming_persistent_service"`
				} `json:"metaDataMap"`
			} `json:"raftMetaData"`
			RaftPort string `json:"raftPort"`
			Version  string `json:"version"`
		} `json:"extendInfo"`
		Address       string `json:"address"`
		FailAccessCnt int    `json:"failAccessCnt"`
	} `json:"servers"`
}

func (c *NacosClient) GetClusterNodes(ip string) (ServersInfo, error) {
	servers := ServersInfo{}
	resp, err := c.httpClient.Get(fmt.Sprintf("http://%s:8848/nacos/v1/ns/operator/servers", ip))
	if err != nil {
		return servers, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return servers, err
	}

	err = json.Unmarshal(body, &servers)
	if err != nil {
		fmt.Printf("%s\n", body)
		return servers, fmt.Errorf(fmt.Sprintf("instance: %s ; %s ;body: %v", ip, err.Error(), string(body)))
	}
	return servers, nil
}

//func (c *CheckClient) getClusterNodesStaus(ip string) (bool, error) {
//	str, err := c.getClusterNodes(ip)
//	if err != nil {
//		return false, err
//	}
//
//}
