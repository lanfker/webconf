package wpa

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type SSIDConf struct {
	SSID     string
	Psk      string
	Priority int64
}

type WPAConf struct {
	Commons string
	SSIDs   []SSIDConf
}

func (w WPAConf) String() string {
	a := fmt.Sprintf("%s\n", strings.TrimSpace(w.Commons))
	for _, v := range w.SSIDs {
		a += fmt.Sprintln(v)
	}
	return a
}

func parseSSIDConf(confStr []byte) (SSIDConf, error) {
	conf := SSIDConf{}
	for k, v := range confStr {
		if v == '\n' {
			confStr[k] = ' '
		}
	}
	idx := bytes.Index(confStr, []byte("network"))
	if idx == -1 {
		return conf, fmt.Errorf("%s:%s", "invalid configuration string", confStr)
	}
	confStr = bytes.TrimLeft(confStr[idx+len("network"):], "\t ")
	idx = bytes.Index(confStr, []byte("="))
	if idx == -1 {
		return conf, fmt.Errorf("%s:%s", "invalid configuration string", confStr)
	}
	confStr = bytes.TrimLeft(confStr[idx+len("="):], " \t")
	idx = bytes.Index(confStr, []byte("{"))
	if idx == -1 {
		return conf, fmt.Errorf("%s:%s", "invalid configuration string", confStr)
	}
	confStr = bytes.TrimLeft(confStr[idx+len("{"):], "\t ")

	idx = bytes.Index(confStr, []byte("}"))
	if idx == -1 {
		return conf, fmt.Errorf("%s:%s", "invalid configuration string", confStr)
	}
	confStr = bytes.TrimLeft(confStr[0:idx], "\t ")
	conf.SSID = parseValue(confStr, []byte("ssid"))
	conf.Psk = parseValue(confStr, []byte("psk"))
	priority := parseValue(confStr, []byte("priority"))
	if len(priority) > 0 {
		conf.Priority, _ = strconv.ParseInt(priority, 10, 64)
	}
	//fmt.Println(conf)
	return conf, nil
}

func parseValue(str, key []byte) string {
	str = bytes.Replace(str, []byte("scan_ssid"), []byte("unknown_1"), 100)
	str = bytes.Replace(str, []byte("key_mgmt"), []byte("unknown_2"), 100)
	idx := bytes.Index(str, key)
	if idx == -1 {
		return ""
	}
	for idx < len(str) {
		if str[idx] == '=' {
			idx++
			break
		}
		idx++
	}

	quoted := false
	for idx < len(str) {
		if !unicode.IsSpace(rune(str[idx])) {
			if str[idx] == '"' || str[idx] == '\'' {
				quoted = true
				idx++
			} else {
				quoted = false
			}
			break
		}
		idx++
	}
	if quoted {
		blank := bytes.Index(str[idx:], []byte(`"`))
		if blank == -1 {
			return string(str[idx:len(str)])
		}
		//fmt.Printf("key: %s, remaining: %s, blank: %d\n", string(key), string(str[idx:]), blank)
		return string(str[idx : blank+idx])
	}
	blank := bytes.Index(str[idx:], []byte(" "))
	if blank == -1 {
		return string(str[idx:len(str)])
	}
	return string(str[idx : blank+idx])

}

func (s SSIDConf) String() string {
	//return fmt.Sprintf("network={\n \tssid=\"%s\"\n\tpsk=\"%s\"\n\tpriority=%d\n}", s.ssid, s.psk, s.priority)
	return fmt.Sprintf(`
network={ 
    ssid="%s" 
    psk="%s" 
    priority=%d 
}`, s.SSID, s.Psk, s.Priority)
}

func ParseFile(filename string) (*WPAConf, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	conf := &WPAConf{}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	inSIIDConf := false
	ssidConfStr := ""
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimLeft(line, " \t") + " "
		if strings.Index(line, "network") == 0 {
			ssidConfStr += line
			inSIIDConf = true
		} else if !inSIIDConf {
			conf.Commons += (line + "\n")
		} else if inSIIDConf && strings.Index(line, "}") != -1 {
			ssidConfStr += line
			inSIIDConf = false
		} else {
			ssidConfStr += line
		}
		if !inSIIDConf && len(ssidConfStr) > 0 {
			ssid, _ := parseSSIDConf([]byte(ssidConfStr))
			conf.SSIDs = append(conf.SSIDs, ssid)
			ssidConfStr = ""
		}
	}
	return conf, nil
}
