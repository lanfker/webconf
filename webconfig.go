package main

import (
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

const (
	wpa_config_file = "wpa_supplicant.conf"
	port            = ":8000"
)

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

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
	log.Fatal(http.ListenAndServe(port, nil))
}

func deleteWifiHandler(w http.ResponseWriter, r *http.Request) {
	ssid := r.URL.Query().Get("ssid")
	wpaconf, _ := wpa.ParseFile(wpa_config_file)
	length := len(wpaconf.SSIDs)
	for k, v := range wpaconf.SSIDs {
		if v.SSID == ssid {
			wpaconf.SSIDs[k] = wpaconf.SSIDs[length-1]
			wpaconf.SSIDs = wpaconf.SSIDs[0 : length-1]
			break
		}
	}
	ioutil.WriteFile(wpa_config_file, []byte(wpaconf.String()), 0755)
	http.Redirect(w, r, "/wifi/", http.StatusFound)
}

func wifiHandler(w http.ResponseWriter, r *http.Request) {
	/*
		tmpl, err := template.ParseFiles("templates/wifi.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	*/
	if r.Method != "POST" {
		wpaconf, _ := wpa.ParseFile(wpa_config_file)
		data := struct {
			Message string
			wpa.WPAConf
		}{
			WPAConf: *wpaconf,
		}
		tmpl.ExecuteTemplate(w, "wifi.html", data)
	} else {
		wpaconf, _ := wpa.ParseFile(wpa_config_file)

		priority, _ := strconv.ParseInt(r.FormValue("priority"), 10, 64)
		ssidconf := wpa.SSIDConf{SSID: r.FormValue("ssid"), Psk: r.FormValue("password"), Priority: priority}
		for k, v := range wpaconf.SSIDs {
			if v.SSID == ssidconf.SSID {
				wpaconf.SSIDs[k].Psk = ssidconf.Psk
				wpaconf.SSIDs[k].Priority = ssidconf.Priority
				ioutil.WriteFile(wpa_config_file, []byte(wpaconf.String()), 0755)
				data := struct {
					Message string
					wpa.WPAConf
				}{
					Message: "SSID configuration updated!",
					WPAConf: *wpaconf,
				}
				tmpl.ExecuteTemplate(w, "wifi.html", data)
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

		ioutil.WriteFile(wpa_config_file, []byte(wpaconf.String()), 0755)
		tmpl.ExecuteTemplate(w, "wifi.html", data)
	}
}

func rebootHandler(w http.ResponseWriter, r *http.Request) {
	/*
		tmpl, err := template.ParseFiles("templates/reboot.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	*/
	tmpl.ExecuteTemplate(w, "reboot.html", nil)
	cmd := exec.Command("reboot")
	err := cmd.Run()
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
			defer filehandle.Close()
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
			defer filehandle.Close()
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

	/*
		tmpl, err := template.ParseFiles("templates/list.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		tmpl.Execute(w, files)
	*/
	tmpl.ExecuteTemplate(w, "list.html", files)

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
	/*
		tmpl, err := template.ParseFiles("templates/activate.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	*/
	conf, err = load(filename)
	conf.Message = fmt.Sprintf("Configuration %s activated", filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "activate.html", conf)

	//tmpl.Execute(w, conf)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/save/"):]
	contentBytes := []byte(r.FormValue("JSONText"))
	fmt.Println(contentBytes)
	if len(contentBytes) == 0 {
		http.Error(w, "No changes available", http.StatusNoContent)
		return
	}
	err := os.Remove(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	f.Write(contentBytes)

	conf, err := load(filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	conf.Message = string("File saved")
	tmpl.ExecuteTemplate(w, "save.html", conf)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/edit/"):]
	/*
		tmpl, err := template.ParseFiles("templates/edit.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	*/
	conf, err := load(filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	tmpl.ExecuteTemplate(w, "edit.html", conf)
}
