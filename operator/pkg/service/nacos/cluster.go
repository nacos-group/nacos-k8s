package nacosClient

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type INacosClient interface {
}

type NacosClient struct {
	logger     log.Logger
	httpClient http.Client
}

type ServersInfo struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Data    []struct {
		IP         string `json:"ip"`
		Port       int    `json:"port"`
		State      string `json:"state"`
		ExtendInfo struct {
			LastRefreshTime int64 `json:"lastRefreshTime"`
			RaftMetaData    struct {
				MetaDataMap struct {
					NamingInstanceMetadata struct {
						Leader          string   `json:"leader"`
						RaftGroupMember []string `json:"raftGroupMember"`
						Term            int      `json:"term"`
					} `json:"naming_instance_metadata"`
					NamingPersistentServiceV2 struct {
						Leader          string   `json:"leader"`
						RaftGroupMember []string `json:"raftGroupMember"`
						Term            int      `json:"term"`
					} `json:"naming_persistent_service_v2"`
					NamingServiceMetadata struct {
						Leader          string   `json:"leader"`
						RaftGroupMember []string `json:"raftGroupMember"`
						Term            int      `json:"term"`
					} `json:"naming_service_metadata"`
				} `json:"metaDataMap"`
			} `json:"raftMetaData"`
			RaftPort       string `json:"raftPort"`
			ReadyToUpgrade bool   `json:"readyToUpgrade"`
			Version        string `json:"version"`
		} `json:"extendInfo"`
		Address       string `json:"address"`
		FailAccessCnt int    `json:"failAccessCnt"`
		Abilities     struct {
			RemoteAbility struct {
				SupportRemoteConnection bool `json:"supportRemoteConnection"`
			} `json:"remoteAbility"`
			ConfigAbility struct {
				SupportRemoteMetrics bool `json:"supportRemoteMetrics"`
			} `json:"configAbility"`
			NamingAbility struct {
				SupportJraft bool `json:"supportJraft"`
			} `json:"namingAbility"`
		} `json:"abilities"`
	} `json:"data"`
}

func (c *NacosClient) GetClusterNodes(ip string, identityKey string, identityValue string) (ServersInfo, error) {
	servers := ServersInfo{}
	//增加支持ipV6 pod状态探测
	client := &http.Client{}
	var err error
	var url string
	if strings.Contains(ip, ":") {
		url = fmt.Sprintf("http://[%s]:8848/nacos/v1/core/cluster/nodes", ip)
	} else {
		url = fmt.Sprintf("http://%s:8848/nacos/v1/core/cluster/nodes", ip)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return servers, err
	}

	if identityKey != "" {
		header := http.Header{
			identityKey: []string{identityValue},
		}
		req.Header = header
	}

	resp, err := client.Do(req)
	if err != nil {
		return servers, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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
