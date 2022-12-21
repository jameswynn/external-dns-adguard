package adguard

import (
	"encoding/json"
	"fmt"
	"github.com/monaco-io/request"
	"github.com/monaco-io/request/response"
	"io"
	"io/ioutil"
	"log"
)

type AdGuardClient struct {
	scheme   string
	baseUrl  string
	username string
	password string
	cache    map[string]string
	log      *log.Logger
}

type adGuardDnsEntry struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

func NewAdGuardClient(scheme string, url string, username string, password string, enableLog bool) *AdGuardClient {
	var logWriter io.Writer
	if enableLog {
		logWriter = log.Writer()
	} else {
		logWriter = ioutil.Discard
	}
	client := AdGuardClient{
		scheme,
		url, username,
		password,
		map[string]string{},
		log.New(logWriter, "<adguard> ", log.LstdFlags|log.Lmsgprefix),
	}
	return &client
}

func (ag *AdGuardClient) httpCall(method string, url string, textBody string) *response.Sugar {
	call := request.Client{
		URL:    fmt.Sprintf("%s://%s%s", ag.scheme, ag.baseUrl, url),
		Method: method,
		BasicAuth: request.BasicAuth{
			Username: ag.username,
			Password: ag.password,
		},
		Header: map[string]string{
			"Content-Type": "application/json",
		},
		String: textBody,
	}
	return call.Send()
}

func (ag *AdGuardClient) CreateEntry(hostname string, ip string) error {
	ag.log.Printf("Create '%s'='%s'\n", hostname, ip)
	body, _ := json.Marshal(adGuardDnsEntry{
		hostname,
		ip,
	})

	resp := ag.httpCall(
		"POST",
		"/control/rewrite/add",
		string(body),
	)
	if !resp.OK() {
		ag.log.Println(resp.Error())
		return fmt.Errorf("failed to create adguard entry: %v", resp.Error())
	} else {
		ag.log.Printf("Response: %s", resp.String())
	}
	err := ag.RefreshEntries()
	if err != nil {
		return err
	}
	return nil
}

func (ag *AdGuardClient) DeleteEntry(hostname string, ip string) error {
	ag.log.Printf("Delete '%s'='%s'\n", hostname, ip)
	body, _ := json.Marshal(adGuardDnsEntry{
		hostname,
		ip,
	})
	resp := ag.httpCall(
		"POST",
		"/control/rewrite/delete",
		string(body),
	)
	if !resp.OK() {
		ag.log.Println(resp.Error())
		return fmt.Errorf("failed to delete adguard entry: %v", resp.Error())
	} else {
		ag.log.Printf("Response: %s", resp.String())
	}
	err := ag.RefreshEntries()
	if err != nil {
		return err
	}
	return nil
}

func (ag *AdGuardClient) RefreshEntries() error {
	ag.log.Printf("Refresh\n")
	resp := ag.httpCall(
		"GET",
		"/control/rewrite/list",
		"",
	)
	if !resp.OK() {
		ag.log.Println(resp.Error())
		return fmt.Errorf("failed to retreive adguard entries: %v", resp.Error())
	}
	var result []adGuardDnsEntry
	err := json.Unmarshal(resp.Bytes(), &result)
	if err != nil {
		ag.log.Println(resp.Error())
		return fmt.Errorf("failed to parse adguard entries: %v", resp.Error())
	}
	entryMap := make(map[string]string)
	for _, entry := range result {
		ag.log.Printf(" - %-30s : %s", entry.Domain, entry.Answer)
		entryMap[entry.Domain] = entry.Answer
	}
	ag.cache = entryMap
	return nil
}

func (ag *AdGuardClient) GetEntries(entries *map[string]string) {
	for k, v := range ag.cache {
		(*entries)[k] = v
	}
}

func (ag *AdGuardClient) GetEntry(hostname string) (string, bool) {
	ip, exists := ag.cache[hostname]
	return ip, exists
}

func (ag *AdGuardClient) EntryInCache(hostname string) bool {
	_, exists := ag.cache[hostname]
	return exists
}
