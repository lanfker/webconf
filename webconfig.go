package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"cvdpweb/wpa"
)

// CvdpConfig dummy
type CvdpConfig struct {
	//
	JSONText template.JS
	Filename string
	Message  string
}

// CvdpFile dummy
type CvdpFile struct {
	Filename     string
	RelativePath string
}

func (c *CvdpConfig) activate(dst string) error {
	err := ioutil.WriteFile(dst, []byte(c.JSONText), 0755)
	if err != nil {
		return err
	}
	return nil
}

//
func load(filename string) (*CvdpConfig, error) {
	conf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &CvdpConfig{Filename: filename, JSONText: template.JS(conf)}, nil
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/files/", fileHandler)
	http.HandleFunc("/", listHandler)
	http.HandleFunc("/list/", listHandler)
	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/save/", saveHandler)
	http.HandleFunc("/activate/", activateHandler)
	http.HandleFunc("/reboot/", rebootHandler)
	http.HandleFunc("/wifi/", wifiHandler)
	http.HandleFunc("/wifi/delete/", deleteWifiHandler)
	log.Fatal(http.ListenAndServe(":80", nil))
}

func deleteWifiHandler(w http.ResponseWriter, r *http.Request) {
	ssid := r.URL.Query().Get("ssid")
	wpaconf, _ := wpa.ParseFile("/etc/wpa_supplicant.conf")
	length := len(wpaconf.SSIDs)
	for k, v := range wpaconf.SSIDs {
		if v.SSID == ssid {
			wpaconf.SSIDs[k] = wpaconf.SSIDs[length-1]
			wpaconf.SSIDs = wpaconf.SSIDs[0 : length-1]
			break
		}
	}
	ioutil.WriteFile("/etc/wpa_supplicant.conf", []byte(wpaconf.String()), 0755)
	http.Redirect(w, r, "/wifi/", http.StatusFound)
}

func wifiHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/wifi.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		wpaconf, _ := wpa.ParseFile("/etc/wpa_supplicant.conf")
		data := struct {
			Message string
			wpa.WPAConf
		}{
			WPAConf: *wpaconf,
		}
		tmpl.Execute(w, data)
	} else {
		wpaconf, _ := wpa.ParseFile("/etc/wpa_supplicant.conf")

		priority, _ := strconv.ParseInt(r.FormValue("priority"), 10, 64)
		ssidconf := wpa.SSIDConf{SSID: r.FormValue("ssid"), Psk: r.FormValue("password"), Priority: priority}
		for k, v := range wpaconf.SSIDs {
			if v.SSID == ssidconf.SSID {
				wpaconf.SSIDs[k].Psk = ssidconf.Psk
				wpaconf.SSIDs[k].Priority = ssidconf.Priority
				ioutil.WriteFile("/etc/wpa_supplicant.conf", []byte(wpaconf.String()), 0755)
				data := struct {
					Message string
					wpa.WPAConf
				}{
					Message: "SSID configuration updated!",
					WPAConf: *wpaconf,
				}
				tmpl.Execute(w, data)
				return
			}
		}

		wpaconf.SSIDs = append(wpaconf.SSIDs, ssidconf)
		data := struct {
			Message string
			wpa.WPAConf
		}{
			Message: "SSID configuration inserted!",
			WPAConf: *wpaconf,
		}

		ioutil.WriteFile("/etc/wpa_supplicant.conf", []byte(wpaconf.String()), 0755)
		tmpl.Execute(w, data)
	}
}

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/reboot.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tmpl.Execute(w, nil)
	cmd := exec.Command("reboot")
	err = cmd.Run()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Path[len("/files/"):]
	for r.Method == "POST" {
		file, handler, err := r.FormFile("fileToUpload")
		if err != nil {
			fmt.Printf("Error Retriving the file, %s\n", err.Error())
			break
		}
		defer file.Close()
		bytes := []byte{}
		_, err = file.Read(bytes)

		if len(base) != 0 {
			filehandle, err := os.OpenFile(base+"/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0755)
			if err != nil {
				fmt.Println("Cannot create the file: ", err.Error())
				break
			}
			_, err = io.Copy(filehandle, file)
			if err != nil {
				fmt.Println("Cannot create the file content: ", err.Error())
				break
			}
		} else {
			filehandle, err := os.OpenFile(handler.Filename, os.O_WRONLY|os.O_CREATE, 0755)
			if err != nil {
				fmt.Println("Cannot create the file: ", err.Error())
				break
			}
			_, err = io.Copy(filehandle, file)
			if err != nil {
				fmt.Println("Cannot create the file content: ", err.Error())
				break
			}
		}
		break
	}

	isDir := false
	if len(base) == 0 {
		base = "."
		isDir = true
	} else {
		file, err := os.Stat(base)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if file.Mode().IsDir() {
			isDir = true
		}
	}
	tmpl, err := template.ParseFiles("templates/files.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if isDir {
		files, err := ioutil.ReadDir(base)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fileWithPath := []CvdpFile{}
		for _, f := range files {
			n := CvdpFile{Filename: f.Name(), RelativePath: base + "/" + f.Name()}
			if base == "." {
				n = CvdpFile{Filename: f.Name(), RelativePath: f.Name()}
			}
			fileWithPath = append(fileWithPath, n)
		}
		tmpl.Execute(w, fileWithPath)
	} else {
		w.Header().Set("Content-Disposition", "attachment;")
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		bytes, err := ioutil.ReadFile(base)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bytes)
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	pattern := "config.json*"
	matches, err := filepath.Glob(pattern)
	files := []CvdpFile{}
	for _, v := range matches {
		files = append(files, CvdpFile{v, "."})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles("templates/list.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tmpl.Execute(w, files)

}

func activateHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/activate/"):]
	conf, err := load(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = conf.activate("config.json")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl, err := template.ParseFiles("templates/activate.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	conf, err = load(filename)
	conf.Message = fmt.Sprintf("Configuration %s activated", filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, conf)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/save/"):]
	var raw map[string]interface{}
	contentBytes := []byte(r.FormValue("JSONText"))
	if len(contentBytes) == 0 {
		http.Error(w, "No changes available", http.StatusNoContent)
		return
	}
	err := json.Unmarshal(contentBytes, &raw)
	if err != nil {
		http.Error(w, "Data posted are not valid JSON", http.StatusPartialContent)
		return
	}

	indented, err := json.MarshalIndent(raw, "", "    ")
	if err != nil {
		http.Error(w, "Convert Post data from JSON to byte stream failed", http.StatusInternalServerError)
		return
	}

	err = ioutil.WriteFile(filename, indented, 0755)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//http.Redirect(w, r, "/save/"+filename, http.StatusFound)
	tmpl, err := template.ParseFiles("templates/save.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	conf, err := load(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	conf.Message = string("File saved")
	tmpl.Execute(w, conf)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/edit/"):]
	tmpl, err := template.ParseFiles("templates/edit.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	conf, err := load(filename)
	tmpl.Execute(w, conf)
}
