package main

import "github.com/hlandau/tftpsrv"
import "net/http"
import "regexp"
import "flag"
import "encoding/json"
import "os"
import "github.com/hlandau/degoutils/log"
import "github.com/hlandau/degoutils/daemon"

var re_valid_fn = regexp.MustCompile("^([a-zA-Z0-9_-][a-zA-Z0-9_. :-]*/)*[a-zA-Z0-9_-][a-zA-Z0-9_. :-]*$")

func validateFilename(fn string) bool {
	return re_valid_fn.MatchString(fn)
}

func handler(req *tftpsrv.Request) error {
	log.Info("GET ", req.Filename)
	defer req.Close()

	addr := req.ClientAddress()
	if !validateFilename(req.Filename) {
		req.WriteError(tftpsrv.ErrFileNotFound, "File not found (invalid filename)")
		log.Error("GET [", addr.IP.String(), "] (bad filename)")
		return nil
	}

	hReq, err := http.NewRequest("GET", settings.HTTP_URL+req.Filename, nil)
	if err != nil {
		return err
	}

	hReq.Header.Add("X-Forwarded-For", addr.IP.String())
	hReq.Header.Add("User-Agent", "tftp2httpd")
	res, err := http.DefaultClient.Do(hReq)
	if err != nil {
		log.Error("GET [", addr.IP.String(), "] ", req.Filename, " -> HTTP Error: ", err)
		return err
	}
	defer res.Body.Close()

	// Don't return error pages.
	if res.StatusCode != 200 {
		req.WriteError(tftpsrv.ErrFileNotFound, "File not found")
		log.Error("GET [", addr.IP.String(), "] ", req.Filename, " -> HTTP Code: ", res.StatusCode)
		return nil
	}

	buf := make([]byte, 512)
	for {
		n, err := res.Body.Read(buf)
		if n > 0 {
			req.Write(buf[0:n])
		}
		if err != nil {
			break
		}
	}

	return nil
}

var settings struct {
	HTTP_URL    string `json:"http_url"`
	TFTP_Listen string `json:"tftp_listen"`
	UID         int    `json:"uid"`
	GID         int    `json:"gid"`
}

func main() {
	cfgPath := flag.String("config-file", "etc/tftp2httpd.json", "JSON configuration file path")
	f_daemon := flag.Bool("daemon", false, "Daemonize (doesn't fork)")
	flag.Parse()

	cfgFile, err := os.Open(*cfgPath)
	log.Fatale(err, "can't open config file")

	json_p := json.NewDecoder(cfgFile)
	err = json_p.Decode(&settings)
	log.Fatale(err, "can't decode configuration file")
	cfgFile.Close()

	s := tftpsrv.Server{
		Addr:        settings.TFTP_Listen,
		ReadHandler: handler,
	}

	err = daemon.Init()
	log.Fatale(err, "can't init daemon")

	if *f_daemon {
		log.OpenSyslog("tftp2httpd")
		err = daemon.Daemonize()
		log.Fatale(err, "can't daemonize")
	}

	err = daemon.DropPrivileges(settings.UID, settings.GID, daemon.EmptyChrootPath)
	log.Fatale(err, "can't drop privileges")

	s.ListenAndServe()
}

// © 2014 Hugo Landau <hlandau@devever.net>    GPLv3 or later
